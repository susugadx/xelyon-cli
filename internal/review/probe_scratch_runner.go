package review

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	if err := validateScratchDirOutsideRepo(e.repoRoot, scratchDir); err != nil {
		result.Status = ReviewProbeBlocked
		result.Error = err.Error()
		return result
	}

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

func validateScratchDirOutsideRepo(repoRoot, scratchDir string) error {
	resolvedScratchDir := filepath.Clean(scratchDir)
	if !filepath.IsAbs(resolvedScratchDir) {
		abs, err := filepath.Abs(resolvedScratchDir)
		if err != nil {
			return newBlockedCommandErrorf("failed to resolve scratch directory %q: %v", scratchDir, err)
		}
		resolvedScratchDir = filepath.Clean(abs)
	}

	insideRepo, err := isPathWithinRepoRoot(repoRoot, resolvedScratchDir)
	if err != nil {
		return newBlockedCommandErrorf("failed to validate scratch directory %q: %v", scratchDir, err)
	}
	if insideRepo {
		return newBlockedCommandErrorf("scratch directory must be outside repository root: %s", resolvedScratchDir)
	}
	return nil
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
	env := append(append([]string(nil), e.baseEnv...),
		scratchEnvRepoRoot+"="+e.repoRoot,
		scratchEnvScratchDir+"="+scratchDir,
	)

	return executeProbeCommand(ctx, probeExecCommand{
		command: cmd.command,
		args:    cmd.args,
		workDir: cmd.workDir,
		env:     env,
	}, timeout, maxOutputBytes)
}
