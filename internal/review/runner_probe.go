package review

import (
	"context"
	"fmt"
)

// runReviewProbesSequentially は plan から作った probe request を順序通りに実行する。
func (r *ReviewRunner) runReviewProbesSequentially(ctx context.Context, probeRequests []ReviewProbeRequest) ([]ReviewProbeResult, error) {
	probeResults := make([]ReviewProbeResult, 0, len(probeRequests))
	for i, probeReq := range probeRequests {
		result, err := r.probeRunner.Run(ctx, probeReq)
		if err != nil {
			return nil, fmt.Errorf("review runner run probe %q: %w", probeReq.ID, err)
		}
		probeResults = append(probeResults, result)
		if result.Status == ReviewProbeMutatedWorktree {
			probeResults = append(probeResults, buildReviewRunnerSkippedProbeResultsAfterMutation(probeRequests[i+1:], result.ID)...)
			break
		}
	}
	return probeResults, nil
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
