package review

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const (
	scratchEnvRepoRoot   = "XELYON_REVIEW_REPO_ROOT"
	scratchEnvScratchDir = "XELYON_REVIEW_SCRATCH_DIR"
)

type scratchOnlyExecutor struct {
	repoRoot  string
	mktemp    func(dir, pattern string) (string, error)
	removeAll func(path string) error
	baseEnv   []string
}

func newScratchOnlyExecutor(repoRoot string) *scratchOnlyExecutor {
	return &scratchOnlyExecutor{
		repoRoot:  repoRoot,
		mktemp:    os.MkdirTemp,
		removeAll: os.RemoveAll,
		baseEnv:   os.Environ(),
	}
}

func (e *scratchOnlyExecutor) run(ctx context.Context, req ReviewProbeRequest) ReviewProbeResult {
	result := ReviewProbeResult{
		ID:     req.ID,
		Mode:   req.Mode,
		Status: ReviewProbePassed,
	}

	scratchDir, err := e.mktemp("", "xelyon-review-scratch-*")
	if err != nil {
		result.Status = ReviewProbeBlocked
		result.Error = fmt.Sprintf("failed to create scratch directory: %v", err)
		return result
	}
	defer func() {
		_ = e.removeAll(scratchDir)
	}()

	normalized, err := e.validateRequest(req, scratchDir)
	if err != nil {
		result.Status = ReviewProbeBlocked
		result.Error = err.Error()
		return result
	}
	result.ID = normalized.id
	result.Mode = normalized.mode

	if err := writeScratchFiles(normalized.files); err != nil {
		result.Status = ReviewProbeBlocked
		result.Error = newBlockedCommandErrorf("failed to write scratch files: %v", err).Error()
		return result
	}

	beforeSnapshot, err := captureWorktreeSnapshot(ctx, e.repoRoot)
	if err != nil {
		applyScratchOnlySnapshotError(&result, "before", err)
		return result
	}

	for _, cmd := range normalized.commands {
		cmdResult := e.executeScratchOnlyCommand(ctx, cmd, normalized.timeout, normalized.maxOutputBytes, normalized.scratchDir)
		if applyScratchOnlyCommandTransition(&result, cmd, cmdResult) {
			break
		}
	}

	afterSnapshot, err := captureWorktreeSnapshot(ctx, e.repoRoot)
	if err != nil {
		applyScratchOnlySnapshotError(&result, "after", err)
		return result
	}

	applyScratchOnlyMutationTransition(&result, diffWorktreeSnapshots(beforeSnapshot, afterSnapshot))
	return result
}

func applyScratchOnlySnapshotError(result *ReviewProbeResult, phase string, err error) {
	result.Status = ReviewProbeBlocked
	result.Error = appendError(result.Error, fmt.Sprintf("failed to capture worktree snapshot %s probe: %v", phase, err))
}

func applyScratchOnlyCommandTransition(result *ReviewProbeResult, cmd scratchOnlyCommand, cmdResult ReviewProbeCommandResult) (stop bool) {
	result.CommandResults = append(result.CommandResults, cmdResult)
	result.OutputTruncated = result.OutputTruncated || cmdResult.OutputTruncated

	switch cmdResult.Status {
	case ReviewProbeTimedOut:
		result.Status = ReviewProbeTimedOut
		result.Error = appendError(result.Error, fmt.Sprintf("probe command timed out: %s", formatProbeCommand(cmd.command, cmd.args)))
		return true
	case ReviewProbeFailed:
		result.Status = ReviewProbeFailed
		result.Error = appendError(result.Error, fmt.Sprintf("probe command failed: %s", formatProbeCommand(cmd.command, cmd.args)))
		return true
	default:
		return false
	}
}

func applyScratchOnlyMutationTransition(result *ReviewProbeResult, mutatedFiles []string) {
	if len(mutatedFiles) == 0 {
		return
	}

	result.MutatedWorktree = true
	result.MutatedFiles = mutatedFiles
	result.Status = ReviewProbeMutatedWorktree
	result.Error = appendError(result.Error, "probe command changed the working tree")
}

func (e *scratchOnlyExecutor) executeScratchOnlyCommand(ctx context.Context, cmd scratchOnlyCommand, timeout time.Duration, maxOutputBytes int64, scratchDir string) ReviewProbeCommandResult {
	result := ReviewProbeCommandResult{
		Command:  cmd.command,
		Args:     append([]string(nil), cmd.args...),
		WorkDir:  cmd.workDir,
		Status:   ReviewProbePassed,
		ExitCode: -1,
	}

	cmdCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		cmdCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	proc := exec.CommandContext(cmdCtx, cmd.command, cmd.args...)
	proc.Dir = cmd.workDir
	proc.Env = append(append([]string(nil), e.baseEnv...),
		scratchEnvRepoRoot+"="+e.repoRoot,
		scratchEnvScratchDir+"="+scratchDir,
	)

	output := newCappedOutput(maxOutputBytes)
	proc.Stdout = output
	proc.Stderr = output

	start := time.Now()
	err := proc.Run()
	result.Duration = time.Since(start)
	result.Output = output.String()
	result.OutputTruncated = output.Truncated()

	if proc.ProcessState != nil {
		result.ExitCode = proc.ProcessState.ExitCode()
	}
	if err == nil {
		return result
	}

	if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
		result.Status = ReviewProbeTimedOut
		result.Error = cmdCtx.Err().Error()
		return result
	}

	result.Status = ReviewProbeFailed
	result.Error = err.Error()
	return result
}
