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

func (e *scratchOnlyExecutor) run(ctx context.Context, req ReviewProbeRequest) (result ReviewProbeResult) {
	// scratch_only は OS sandbox ではない。
	// env/cache/tmp/home の隔離と request limits により漏洩・副作用を減らすが、
	// 実行中の network / 任意 process 起動 / 任意 file access を OS レベルでは防がない。
	result = newScratchOnlyProbeResult(req)

	scratchDir, err := e.mktemp("", "xelyon-review-scratch-*")
	if err != nil {
		blockScratchOnlyResult(&result, fmt.Sprintf("failed to create scratch directory: %v", err))
		return result
	}
	defer func() {
		e.cleanupScratchDir(&result, scratchDir)
	}()
	runtime, err := e.prepareRuntime(req, scratchDir)
	if err != nil {
		blockScratchOnlyResult(&result, err.Error())
		return result
	}
	result.ID = runtime.request.id
	result.Mode = runtime.request.mode

	if err := writeScratchFiles(runtime.request.files); err != nil {
		blockScratchOnlyResult(&result, newBlockedCommandErrorf("failed to write scratch files: %v", err).Error())
		return result
	}

	beforeSnapshot, err := captureWorktreeSnapshot(ctx, e.repoRoot)
	if err != nil {
		applyScratchOnlySnapshotError(&result, "before", err)
		return result
	}

	for _, cmd := range runtime.request.commands {
		// scratch_only は OS sandbox ではない。env/cache/limits で副作用と漏洩を減らす。
		cmdResult := e.executeScratchOnlyCommand(ctx, cmd, runtime.request.timeout, runtime.request.maxOutputBytes, runtime.env)
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

func newScratchOnlyProbeResult(req ReviewProbeRequest) ReviewProbeResult {
	return ReviewProbeResult{
		ID:     req.ID,
		Mode:   req.Mode,
		Status: ReviewProbePassed,
	}
}

func blockScratchOnlyResult(result *ReviewProbeResult, message string) {
	result.Status = ReviewProbeBlocked
	result.Error = message
}

func (e *scratchOnlyExecutor) cleanupScratchDir(result *ReviewProbeResult, scratchDir string) {
	if err := e.removeAll(scratchDir); err != nil {
		appendScratchCleanupError(result, scratchDir, err)
	}
}

func appendScratchCleanupError(result *ReviewProbeResult, scratchDir string, err error) {
	result.Error = appendError(result.Error, fmt.Sprintf("failed to remove scratch directory %q: %v", scratchDir, err))
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

func (e *scratchOnlyExecutor) executeScratchOnlyCommand(ctx context.Context, cmd scratchOnlyCommand, timeout time.Duration, maxOutputBytes int64, env []string) ReviewProbeCommandResult {
	return executeProbeCommand(ctx, probeExecCommand{
		command:     cmd.command,
		commandPath: cmd.commandPath,
		args:        cmd.args,
		workDir:     cmd.workDir,
		env:         env,
	}, timeout, maxOutputBytes)
}
