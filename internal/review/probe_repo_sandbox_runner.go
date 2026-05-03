package review

import (
	"context"
	"fmt"
	"os"
	"time"
)

type repoSandboxExecutor struct {
	repoRoot  string
	mktemp    func(dir, pattern string) (string, error)
	removeAll func(path string) error
	baseEnv   []string
}

func newRepoSandboxExecutor(repoRoot string) *repoSandboxExecutor {
	return &repoSandboxExecutor{
		repoRoot:  repoRoot,
		mktemp:    os.MkdirTemp,
		removeAll: os.RemoveAll,
		baseEnv:   os.Environ(),
	}
}

func (e *repoSandboxExecutor) run(ctx context.Context, req ReviewProbeRequest) (result ReviewProbeResult) {
	// repo_sandbox は OS sandbox ではない。repo copy、env/cache/tmp/home の隔離、
	// request policy により元 repo への通常の副作用を避ける。
	result = newRepoSandboxProbeResult(req)

	sandboxRoot, err := e.mktemp("", "xelyon-review-sandbox-*")
	if err != nil {
		blockRepoSandboxResult(&result, fmt.Sprintf("failed to create repo_sandbox root: %v", err))
		return result
	}
	defer func() {
		e.cleanupRepoSandboxRoot(&result, sandboxRoot)
	}()

	runtime, err := e.prepareRuntime(req, sandboxRoot)
	if err != nil {
		blockRepoSandboxResult(&result, err.Error())
		return result
	}
	result.ID = runtime.request.id
	result.Mode = runtime.request.mode

	if err := writeRepoSandboxFiles(runtime.request.files); err != nil {
		blockRepoSandboxResult(&result, newBlockedCommandErrorf("failed to write repo_sandbox generated files: %v", err).Error())
		return result
	}

	beforeSnapshot, err := captureWorktreeSnapshot(ctx, e.repoRoot)
	if err != nil {
		applyRepoSandboxSnapshotError(&result, "before", err)
		return result
	}

	for _, cmd := range runtime.request.commands {
		cmdResult := e.executeRepoSandboxCommand(ctx, cmd, runtime.request.timeout, runtime.request.maxOutputBytes, runtime.env)
		if applyRepoSandboxCommandTransition(&result, cmd, cmdResult) {
			break
		}
	}

	afterSnapshot, err := captureWorktreeSnapshot(ctx, e.repoRoot)
	if err != nil {
		applyRepoSandboxSnapshotError(&result, "after", err)
		return result
	}

	applyRepoSandboxMutationTransition(&result, diffWorktreeSnapshots(beforeSnapshot, afterSnapshot))
	return result
}

func newRepoSandboxProbeResult(req ReviewProbeRequest) ReviewProbeResult {
	return newIsolatedProbeResult(req)
}

func blockRepoSandboxResult(result *ReviewProbeResult, message string) {
	blockIsolatedProbeResult(result, message)
}

func (e *repoSandboxExecutor) cleanupRepoSandboxRoot(result *ReviewProbeResult, sandboxRoot string) {
	if err := e.removeAll(sandboxRoot); err != nil {
		appendRepoSandboxCleanupError(result, sandboxRoot, err)
	}
}

func appendRepoSandboxCleanupError(result *ReviewProbeResult, sandboxRoot string, err error) {
	appendIsolatedCleanupError(result, "repo_sandbox root", sandboxRoot, err)
}

func applyRepoSandboxSnapshotError(result *ReviewProbeResult, phase string, err error) {
	applyIsolatedProbeSnapshotError(result, phase, err)
}

func applyRepoSandboxCommandTransition(result *ReviewProbeResult, cmd repoSandboxCommand, cmdResult ReviewProbeCommandResult) (stop bool) {
	return applyIsolatedProbeCommandTransition(result, cmd.command, cmd.args, cmdResult)
}

func applyRepoSandboxMutationTransition(result *ReviewProbeResult, mutatedFiles []string) {
	applyIsolatedProbeMutationTransition(result, mutatedFiles)
}

func (e *repoSandboxExecutor) executeRepoSandboxCommand(ctx context.Context, cmd repoSandboxCommand, timeout time.Duration, maxOutputBytes int64, env []string) ReviewProbeCommandResult {
	return executeProbeCommand(ctx, probeExecCommand{
		command:     cmd.command,
		commandPath: cmd.commandPath,
		args:        cmd.args,
		workDir:     cmd.workDir,
		env:         env,
	}, timeout, maxOutputBytes)
}
