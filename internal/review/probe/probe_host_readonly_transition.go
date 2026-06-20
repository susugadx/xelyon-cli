package probe

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

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
