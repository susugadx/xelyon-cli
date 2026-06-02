package probe

import (
	"context"
	"fmt"
	"os"
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
	// scratch_only は process sandbox 内で、repo を read-only、scratch を read-write として実行する。
	// 生成ファイルと runtime cache/tmp/home は scratch 側に閉じる。
	result = newScratchOnlyProbeResult(req)

	scratchDir, err := e.mktemp("", reviewProbeScratchTempPattern)
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
		cmdResult := e.executeScratchOnlyCommand(ctx, cmd, runtime.request.timeout, runtime.request.maxOutputBytes, runtime.env, runtime.sandbox)
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
	return newIsolatedProbeResult(req)
}

func blockScratchOnlyResult(result *ReviewProbeResult, message string) {
	blockIsolatedProbeResult(result, message)
}

func (e *scratchOnlyExecutor) cleanupScratchDir(result *ReviewProbeResult, scratchDir string) {
	if err := e.removeAll(scratchDir); err != nil {
		appendScratchCleanupError(result, scratchDir, err)
	}
}

func appendScratchCleanupError(result *ReviewProbeResult, scratchDir string, err error) {
	appendIsolatedCleanupError(result, "scratch directory", scratchDir, err)
}

func validateScratchDirOutsideRepo(repoRoot, scratchDir string) error {
	return validateIsolatedRootOutsideRepo(repoRoot, scratchDir, "scratch")
}

func applyScratchOnlySnapshotError(result *ReviewProbeResult, phase string, err error) {
	applyIsolatedProbeSnapshotError(result, phase, err)
}

func applyScratchOnlyCommandTransition(result *ReviewProbeResult, cmd scratchOnlyCommand, cmdResult ReviewProbeCommandResult) (stop bool) {
	return applyIsolatedProbeCommandTransition(result, cmd.command, cmd.args, cmdResult)
}

func applyScratchOnlyMutationTransition(result *ReviewProbeResult, mutatedFiles []string) {
	applyIsolatedProbeMutationTransition(result, mutatedFiles)
}

func (e *scratchOnlyExecutor) executeScratchOnlyCommand(ctx context.Context, cmd scratchOnlyCommand, timeout time.Duration, maxOutputBytes int64, env []string, sandbox probeProcessSandbox) ReviewProbeCommandResult {
	return executeProbeCommand(ctx, probeExecCommand{
		command:     cmd.command,
		commandPath: cmd.commandPath,
		args:        cmd.args,
		workDir:     cmd.workDir,
		env:         env,
		sandbox:     sandbox,
	}, timeout, maxOutputBytes)
}
