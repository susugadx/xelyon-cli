package review

import reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"

// canonicalizeReviewProbeResultMutationOutcome は mutation outcome の内部表現を揃える。
// status と flag のどちらかが mutation を示す場合、両方を mutation として扱う。
func canonicalizeReviewProbeResultMutationOutcome(result ReviewProbeResult) ReviewProbeResult {
	return reviewprobe.CanonicalizeReviewProbeResultMutationOutcome(result)
}

func canonicalizeReviewProbeSummaryMutationOutcome(summary ReviewProbeSummary) ReviewProbeSummary {
	if !isReviewProbeSummaryMutationOutcome(summary) {
		return summary
	}
	summary.Status = ReviewProbeMutatedWorktree
	summary.MutatedWorktree = true
	return summary
}

func canonicalizeReviewProbeSummaryMutationOutcomes(summaries []ReviewProbeSummary) {
	for i := range summaries {
		summaries[i] = canonicalizeReviewProbeSummaryMutationOutcome(summaries[i])
	}
}

func isReviewProbeResultMutationOutcome(result ReviewProbeResult) bool {
	return reviewprobe.IsReviewProbeResultMutationOutcome(result)
}

func isReviewProbeSummaryMutationOutcome(summary ReviewProbeSummary) bool {
	return isReviewProbeMutationOutcome(summary.Status, summary.MutatedWorktree)
}

func isReviewProbeMutationOutcome(status ReviewProbeStatus, mutatedWorktree bool) bool {
	return reviewprobe.IsReviewProbeMutationOutcome(status, mutatedWorktree)
}
