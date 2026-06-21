package probe

import (
	"context"
	"fmt"
	"os"
)

type hostReadOnlyExecutor struct {
	repoRoot  string
	mktemp    func(dir, pattern string) (string, error)
	removeAll func(path string) error
	baseEnv   []string
}

func newHostReadOnlyExecutor(repoRoot string) *hostReadOnlyExecutor {
	return &hostReadOnlyExecutor{
		repoRoot:  repoRoot,
		mktemp:    os.MkdirTemp,
		removeAll: os.RemoveAll,
		baseEnv:   os.Environ(),
	}
}

func (e *hostReadOnlyExecutor) run(ctx context.Context, req ReviewProbeRequest) (result ReviewProbeResult) {
	reducer := newHostReadOnlyResultReducer(req)

	runtimeRoot, err := e.mktemp("", reviewProbeHostReadOnlyTempPattern)
	if err != nil {
		reducer.applyValidationError(fmt.Errorf("failed to create host_readonly runtime root: %w", err))
		return reducer.resultValue()
	}
	defer func() {
		e.cleanupHostReadOnlyRuntimeRoot(&reducer.result, runtimeRoot)
		result = reducer.resultValue()
	}()

	runtime, err := e.prepareRuntime(req, runtimeRoot)
	if err != nil {
		reducer.applyValidationError(err)
		return reducer.resultValue()
	}
	reducer.applyNormalizedRequest(runtime.request)

	beforeSnapshot, err := captureWorktreeSnapshot(ctx, e.repoRoot)
	if err != nil {
		reducer.applySnapshotError("before", err)
		return reducer.resultValue()
	}

	for _, cmd := range runtime.request.commands {
		cmdResult := executeHostReadOnlyCommand(ctx, cmd, runtime.request.timeout, runtime.request.maxOutputBytes, runtime.env, runtime.sandbox)
		if reducer.applyCommandResult(cmd, cmdResult) {
			break
		}
	}

	afterSnapshot, err := captureWorktreeSnapshot(ctx, e.repoRoot)
	if err != nil {
		reducer.applySnapshotError("after", err)
		return reducer.resultValue()
	}

	reducer.applyMutation(diffWorktreeSnapshots(beforeSnapshot, afterSnapshot))
	return reducer.resultValue()
}
