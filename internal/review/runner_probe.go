package review

import (
	"context"
	"fmt"
	"time"

	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
)

// runReviewProbesSequentially は plan から作った probe request を順序通りに実行する。
func (r *ReviewRunner) runReviewProbesSequentially(ctx context.Context, probeRequests []reviewprobe.ReviewProbeRequest) ([]reviewprobe.ReviewProbeResult, error) {
	probeResults := make([]reviewprobe.ReviewProbeResult, 0, len(probeRequests))
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
			skippedResults := reviewprobe.BuildSkippedProbeResultsAfterMutation(skippedProbeRequests, result.ID)
			for skippedIndex, skipped := range skippedResults {
				r.emitProbeResultProgress(reviewProgressProbeScopeForRequest(skippedProbeRequests[skippedIndex]), skipped, time.Time{})
			}
			probeResults = append(probeResults, skippedResults...)
			break
		}
	}
	return probeResults, nil
}

func (r *ReviewRunner) emitProbeResultProgress(progressScope reviewProgressProbeScope, result reviewprobe.ReviewProbeResult, startedAt time.Time) {
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
