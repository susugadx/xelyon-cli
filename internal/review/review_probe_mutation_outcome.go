package review

import (
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

// canonicalizeReviewProbeResultMutationOutcome は mutation outcome の内部表現を揃える。
// status と flag のどちらかが mutation を示す場合、両方を mutation として扱う。
func canonicalizeReviewProbeResultMutationOutcome(result ReviewProbeResult) ReviewProbeResult {
	return reviewprobe.CanonicalizeReviewProbeResultMutationOutcome(result)
}

func canonicalizeReviewProbeSummaryMutationOutcome(summary ReviewProbeSummary) ReviewProbeSummary {
	return reviewreport.CanonicalizeReviewProbeSummaryMutationOutcome(summary)
}

func isReviewProbeResultMutationOutcome(result ReviewProbeResult) bool {
	return reviewprobe.IsReviewProbeResultMutationOutcome(result)
}
