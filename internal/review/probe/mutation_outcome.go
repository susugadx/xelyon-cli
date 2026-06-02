package probe

import "fmt"

// CanonicalizeReviewProbeResultMutationOutcome は mutation outcome の内部表現を揃える。
// status と flag のどちらかが mutation を示す場合、両方を mutation として扱う。
func CanonicalizeReviewProbeResultMutationOutcome(result ReviewProbeResult) ReviewProbeResult {
	if !IsReviewProbeResultMutationOutcome(result) {
		return result
	}
	result.Status = ReviewProbeMutatedWorktree
	result.MutatedWorktree = true
	return result
}

// IsReviewProbeResultMutationOutcome は probe result が worktree mutation outcome かを返す。
func IsReviewProbeResultMutationOutcome(result ReviewProbeResult) bool {
	return IsReviewProbeMutationOutcome(result.Status, result.MutatedWorktree)
}

// IsReviewProbeMutationOutcome は status/flag の組み合わせが mutation outcome かを返す。
func IsReviewProbeMutationOutcome(status ReviewProbeStatus, mutatedWorktree bool) bool {
	return status == ReviewProbeMutatedWorktree || mutatedWorktree
}

// BuildSkippedProbeResultsAfterMutation は mutation 検出後に未実行 probe を blocked result として固定する。
func BuildSkippedProbeResultsAfterMutation(probeRequests []ReviewProbeRequest, mutatedProbeID string) []ReviewProbeResult {
	if len(probeRequests) == 0 {
		return nil
	}

	results := make([]ReviewProbeResult, 0, len(probeRequests))
	for _, probeReq := range probeRequests {
		results = append(results, ReviewProbeResult{
			ID:     probeReq.ID,
			Mode:   probeReq.Mode,
			Status: ReviewProbeBlocked,
			Error:  ProbeSkippedAfterMutationError(mutatedProbeID),
		})
	}
	return results
}

// ProbeSkippedAfterMutationError は mutation 後に skip された probe の固定 error 文を返す。
func ProbeSkippedAfterMutationError(mutatedProbeID string) string {
	return fmt.Sprintf("probe skipped because probe %q mutated the working tree", mutatedProbeID)
}
