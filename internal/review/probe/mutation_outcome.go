package probe

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
