package review

import (
	"context"
	"fmt"
	"time"
)

// runReviewProbesSequentially は plan から作った probe request を順序通りに実行する。
func (r *ReviewRunner) runReviewProbesSequentially(ctx context.Context, probeRequests []ReviewProbeRequest) ([]ReviewProbeResult, error) {
	probeResults := make([]ReviewProbeResult, 0, len(probeRequests))
	for i, probeReq := range probeRequests {
		startedAt := time.Now()
		progressScope := reviewProgressProbeScopeForRequest(probeReq)
		r.emitProgress(ReviewProgressEvent{
			ID:     progressScope.startedID,
			Phase:  ReviewProgressPhaseProbe,
			Status: ReviewProgressRunning,
			Label:  reviewProgressProbeLabel(probeReq.Mode),
			Detail: reviewProgressProbeDetail(probeReq),
		})

		result, err := r.probeRunner.Run(ctx, probeReq)
		if err != nil {
			r.emitProgress(ReviewProgressEvent{
				ID:       progressScope.startedID,
				Phase:    ReviewProgressPhaseProbe,
				Status:   ReviewProgressError,
				Label:    reviewProgressProbeLabel(probeReq.Mode),
				Detail:   truncateReviewProgressDetail(err.Error()),
				Duration: reviewProgressDuration(startedAt),
			})
			return nil, fmt.Errorf("review runner run probe %q: %w", probeReq.ID, err)
		}
		result = canonicalizeReviewProbeResultMutationOutcome(result)
		r.emitProbeResultProgress(progressScope, result, startedAt)
		probeResults = append(probeResults, result)
		if isReviewProbeResultMutationOutcome(result) {
			skippedProbeRequests := probeRequests[i+1:]
			skippedResults := buildReviewRunnerSkippedProbeResultsAfterMutation(skippedProbeRequests, result.ID)
			for skippedIndex, skipped := range skippedResults {
				r.emitProbeResultProgress(reviewProgressProbeScopeForRequest(skippedProbeRequests[skippedIndex]), skipped, time.Time{})
			}
			probeResults = append(probeResults, skippedResults...)
			break
		}
	}
	return probeResults, nil
}

func (r *ReviewRunner) emitProbeResultProgress(progressScope reviewProgressProbeScope, result ReviewProbeResult, startedAt time.Time) {
	if len(result.CommandResults) == 0 {
		r.emitProgress(ReviewProgressEvent{
			ID:       progressScope.eventID(-1),
			Phase:    ReviewProgressPhaseProbe,
			Status:   reviewProgressStatusForProbeStatus(result.Status),
			Label:    reviewProgressProbeLabel(result.Mode),
			Detail:   reviewProgressProbeResultDetail(result),
			Duration: reviewProgressDuration(startedAt),
		})
		return
	}

	for index, command := range result.CommandResults {
		status := command.Status
		if status == "" {
			status = result.Status
		}
		r.emitProgress(ReviewProgressEvent{
			ID:       progressScope.eventID(index),
			Phase:    ReviewProgressPhaseProbe,
			Status:   reviewProgressStatusForProbeStatus(status),
			Label:    reviewProgressProbeLabel(result.Mode),
			Detail:   reviewProgressProbeCommandDetail(command),
			Duration: command.Duration,
		})
	}
}

func buildReviewRunnerSkippedProbeResultsAfterMutation(probeRequests []ReviewProbeRequest, mutatedProbeID string) []ReviewProbeResult {
	if len(probeRequests) == 0 {
		return nil
	}

	results := make([]ReviewProbeResult, 0, len(probeRequests))
	for _, probeReq := range probeRequests {
		results = append(results, ReviewProbeResult{
			ID:     probeReq.ID,
			Mode:   probeReq.Mode,
			Status: ReviewProbeBlocked,
			Error:  reviewRunnerProbeSkippedAfterMutationError(mutatedProbeID),
		})
	}
	return results
}

func reviewRunnerProbeSkippedAfterMutationError(mutatedProbeID string) string {
	return fmt.Sprintf("probe skipped because probe %q mutated the working tree", mutatedProbeID)
}
