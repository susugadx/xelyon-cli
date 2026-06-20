package probe

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

const (
	defaultReviewProbeTimeout        = 30 * time.Second
	defaultReviewProbeMaxOutputBytes = 64 * 1024
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

type hostReadOnlyCommand struct {
	command     string
	commandPath string
	args        []string
	workDir     string
}

type hostReadOnlyRequest struct {
	id             string
	mode           domain.ReviewProbeMode
	timeout        time.Duration
	maxOutputBytes int64
	commands       []hostReadOnlyCommand
}

type hostReadOnlyRuntime struct {
	request hostReadOnlyRequest
	env     []string
	sandbox probeProcessSandbox
}

type hostReadOnlyResultReducer struct {
	result ReviewProbeResult
}

func newHostReadOnlyResultReducer(req ReviewProbeRequest) *hostReadOnlyResultReducer {
	return &hostReadOnlyResultReducer{
		result: ReviewProbeResult{
			ID:     req.ID,
			Mode:   req.Mode,
			Status: domain.ReviewProbePassed,
		},
	}
}

func (r *hostReadOnlyResultReducer) applyValidationError(err error) {
	r.result.Status = domain.ReviewProbeBlocked
	r.result.Error = err.Error()
}

func (r *hostReadOnlyResultReducer) applyNormalizedRequest(req hostReadOnlyRequest) {
	r.result.ID = req.id
	r.result.Mode = req.mode
}

func (r *hostReadOnlyResultReducer) applySnapshotError(phase string, err error) {
	r.result.Status = domain.ReviewProbeBlocked
	r.result.Error = appendError(r.result.Error, fmt.Sprintf("failed to capture worktree snapshot %s probe: %v", phase, err))
}

func (r *hostReadOnlyResultReducer) applyCommandResult(cmd hostReadOnlyCommand, cmdResult ReviewProbeCommandResult) (stop bool) {
	r.result.CommandResults = append(r.result.CommandResults, cmdResult)
	r.result.OutputTruncated = r.result.OutputTruncated || cmdResult.OutputTruncated

	nextStatus, message, stop := buildProbeCommandTransition(cmdResult.Status, cmd.command, cmd.args)
	if !stop {
		return false
	}
	r.result.Status = nextStatus
	r.result.Error = appendError(r.result.Error, message)
	return true
}

func (r *hostReadOnlyResultReducer) applyMutation(mutatedFiles []string) {
	if len(mutatedFiles) == 0 {
		return
	}

	r.result.MutatedWorktree = true
	r.result.MutatedFiles = mutatedFiles
	r.result.Status = domain.ReviewProbeMutatedWorktree
	r.result.Error = appendError(r.result.Error, "probe command changed the working tree")
}

func (r *hostReadOnlyResultReducer) resultValue() ReviewProbeResult {
	return r.result
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

func (e *hostReadOnlyExecutor) prepareRuntime(req ReviewProbeRequest, runtimeRoot string) (hostReadOnlyRuntime, error) {
	if err := validateHostReadOnlyRuntimeRootOutsideRepo(e.repoRoot, runtimeRoot); err != nil {
		return hostReadOnlyRuntime{}, err
	}

	dirs, err := prepareHostReadOnlyRuntimeDirs(runtimeRoot)
	if err != nil {
		return hostReadOnlyRuntime{}, newBlockedCommandErrorf("failed to prepare host_readonly runtime directories: %v", err)
	}
	env := buildHostReadOnlyEnv(e.baseEnv, e.repoRoot, dirs)

	normalized, err := e.validateRequest(req, env, dirs.RootDir)
	if err != nil {
		return hostReadOnlyRuntime{}, err
	}

	sandboxReadOnlyBinds := probeGoModuleCacheReadOnlyBinds(e.baseEnv, e.repoRoot, dirs.GoModCacheDir)
	sandbox, err := newHostReadOnlyProcessSandbox(e.repoRoot, dirs.RootDir, sandboxReadOnlyBinds...)
	if err != nil {
		return hostReadOnlyRuntime{}, err
	}

	return hostReadOnlyRuntime{
		request: normalized,
		env:     env,
		sandbox: sandbox,
	}, nil
}

func (e *hostReadOnlyExecutor) cleanupHostReadOnlyRuntimeRoot(result *ReviewProbeResult, runtimeRoot string) {
	if err := e.removeAll(runtimeRoot); err != nil {
		appendHostReadOnlyCleanupError(result, runtimeRoot, err)
	}
}

func appendHostReadOnlyCleanupError(result *ReviewProbeResult, runtimeRoot string, err error) {
	appendIsolatedCleanupError(result, "host_readonly runtime root", runtimeRoot, err)
}

func validateHostReadOnlyRuntimeRootOutsideRepo(repoRoot, runtimeRoot string) error {
	return validateIsolatedRootOutsideRepo(repoRoot, runtimeRoot, "host_readonly runtime")
}
