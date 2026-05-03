package review

import (
	"context"
	"fmt"
	"os"
	"time"
)

const (
	defaultReviewProbeTimeout        = 30 * time.Second
	defaultReviewProbeMaxOutputBytes = 64 * 1024
)

type hostReadOnlyExecutor struct {
	repoRoot string
	baseEnv  []string
}

func newHostReadOnlyExecutor(repoRoot string) *hostReadOnlyExecutor {
	return &hostReadOnlyExecutor{
		repoRoot: repoRoot,
		baseEnv:  os.Environ(),
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
	mode           ReviewProbeMode
	timeout        time.Duration
	maxOutputBytes int64
	commands       []hostReadOnlyCommand
}

type hostReadOnlyResultReducer struct {
	result ReviewProbeResult
}

func newHostReadOnlyResultReducer(req ReviewProbeRequest) *hostReadOnlyResultReducer {
	return &hostReadOnlyResultReducer{
		result: ReviewProbeResult{
			ID:     req.ID,
			Mode:   req.Mode,
			Status: ReviewProbePassed,
		},
	}
}

func (r *hostReadOnlyResultReducer) applyValidationError(err error) {
	r.result.Status = ReviewProbeBlocked
	r.result.Error = err.Error()
}

func (r *hostReadOnlyResultReducer) applyNormalizedRequest(req hostReadOnlyRequest) {
	r.result.ID = req.id
	r.result.Mode = req.mode
}

func (r *hostReadOnlyResultReducer) applySnapshotError(phase string, err error) {
	r.result.Status = ReviewProbeBlocked
	r.result.Error = appendError(r.result.Error, fmt.Sprintf("failed to capture worktree snapshot %s probe: %v", phase, err))
}

func (r *hostReadOnlyResultReducer) applyCommandResult(cmd hostReadOnlyCommand, cmdResult ReviewProbeCommandResult) (stop bool) {
	r.result.CommandResults = append(r.result.CommandResults, cmdResult)
	r.result.OutputTruncated = r.result.OutputTruncated || cmdResult.OutputTruncated

	switch cmdResult.Status {
	case ReviewProbeTimedOut:
		r.result.Status = ReviewProbeTimedOut
		r.result.Error = appendError(r.result.Error, fmt.Sprintf("probe command timed out: %s", formatProbeCommand(cmd.command, cmd.args)))
		return true
	case ReviewProbeFailed:
		r.result.Status = ReviewProbeFailed
		r.result.Error = appendError(r.result.Error, fmt.Sprintf("probe command failed: %s", formatProbeCommand(cmd.command, cmd.args)))
		return true
	default:
		return false
	}
}

func (r *hostReadOnlyResultReducer) applyMutation(mutatedFiles []string) {
	if len(mutatedFiles) == 0 {
		return
	}

	r.result.MutatedWorktree = true
	r.result.MutatedFiles = mutatedFiles
	r.result.Status = ReviewProbeMutatedWorktree
	r.result.Error = appendError(r.result.Error, "probe command changed the working tree")
}

func (r *hostReadOnlyResultReducer) resultValue() ReviewProbeResult {
	return r.result
}

func (e *hostReadOnlyExecutor) run(ctx context.Context, req ReviewProbeRequest) ReviewProbeResult {
	reducer := newHostReadOnlyResultReducer(req)

	normalized, err := e.validateRequest(req)
	if err != nil {
		reducer.applyValidationError(err)
		return reducer.resultValue()
	}
	reducer.applyNormalizedRequest(normalized)

	beforeSnapshot, err := captureWorktreeSnapshot(ctx, e.repoRoot)
	if err != nil {
		reducer.applySnapshotError("before", err)
		return reducer.resultValue()
	}

	for _, cmd := range normalized.commands {
		cmdResult := executeHostReadOnlyCommand(ctx, cmd, normalized.timeout, normalized.maxOutputBytes)
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
