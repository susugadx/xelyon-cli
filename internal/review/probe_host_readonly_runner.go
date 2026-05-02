package review

import (
	"context"
	"fmt"
	"time"
)

const (
	defaultReviewProbeTimeout        = 30 * time.Second
	defaultReviewProbeMaxOutputBytes = 64 * 1024
)

type hostReadOnlyExecutor struct {
	repoRoot string
}

func newHostReadOnlyExecutor(repoRoot string) *hostReadOnlyExecutor {
	return &hostReadOnlyExecutor{repoRoot: repoRoot}
}

type hostReadOnlyCommand struct {
	command string
	args    []string
	workDir string
}

type hostReadOnlyRequest struct {
	id             string
	mode           ReviewProbeMode
	timeout        time.Duration
	maxOutputBytes int64
	commands       []hostReadOnlyCommand
}

func (e *hostReadOnlyExecutor) run(ctx context.Context, req ReviewProbeRequest) ReviewProbeResult {
	result := ReviewProbeResult{
		ID:     req.ID,
		Mode:   req.Mode,
		Status: ReviewProbePassed,
	}

	normalized, err := e.validateRequest(req)
	if err != nil {
		result.Status = ReviewProbeBlocked
		result.Error = err.Error()
		return result
	}

	result.ID = normalized.id
	result.Mode = normalized.mode

	beforeSnapshot, err := captureWorktreeSnapshot(ctx, e.repoRoot)
	if err != nil {
		applyHostReadOnlySnapshotError(&result, "before", err)
		return result
	}

	for _, cmd := range normalized.commands {
		cmdResult := executeHostReadOnlyCommand(ctx, cmd, normalized.timeout, normalized.maxOutputBytes)
		if applyHostReadOnlyCommandTransition(&result, cmd, cmdResult) {
			break
		}
	}

	afterSnapshot, err := captureWorktreeSnapshot(ctx, e.repoRoot)
	if err != nil {
		applyHostReadOnlySnapshotError(&result, "after", err)
		return result
	}

	applyHostReadOnlyMutationTransition(&result, diffWorktreeSnapshots(beforeSnapshot, afterSnapshot))

	return result
}

func applyHostReadOnlySnapshotError(result *ReviewProbeResult, phase string, err error) {
	result.Status = ReviewProbeBlocked
	result.Error = appendError(result.Error, fmt.Sprintf("failed to capture worktree snapshot %s probe: %v", phase, err))
}

func applyHostReadOnlyCommandTransition(result *ReviewProbeResult, cmd hostReadOnlyCommand, cmdResult ReviewProbeCommandResult) (stop bool) {
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

func applyHostReadOnlyMutationTransition(result *ReviewProbeResult, mutatedFiles []string) {
	if len(mutatedFiles) == 0 {
		return
	}

	result.MutatedWorktree = true
	result.MutatedFiles = mutatedFiles
	result.Status = ReviewProbeMutatedWorktree
	result.Error = appendError(result.Error, "probe command changed the working tree")
}
