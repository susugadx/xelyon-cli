package review

import reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"

// BuildReviewProbeSummary は probe 結果を report 用要約へ変換する。
func BuildReviewProbeSummary(result ReviewProbeResult) ReviewProbeSummary {
	return reviewprobe.BuildReviewProbeSummary(result)
}

// BuildReviewProbeSummaries は複数 probe 結果をまとめて要約する。
func BuildReviewProbeSummaries(results []ReviewProbeResult) []ReviewProbeSummary {
	return reviewprobe.BuildReviewProbeSummaries(results)
}
