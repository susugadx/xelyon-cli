package probe

import "github.com/susugadx/xelyon-cli/internal/review/report"

// BuildReviewProbeSummary は probe 結果を report 用要約へ変換する。
func BuildReviewProbeSummary(result ReviewProbeResult) report.ReviewProbeSummary {
	summary := report.ReviewProbeSummary{
		ProbeID:         result.ID,
		Mode:            result.Mode,
		Status:          result.Status,
		MutatedWorktree: result.MutatedWorktree,
		MutatedFiles:    append([]string(nil), result.MutatedFiles...),
		OutputTruncated: result.OutputTruncated,
		Error:           result.Error,
		Commands:        make([]report.ReviewProbeCommandSummary, 0, len(result.CommandResults)),
	}
	for _, commandResult := range result.CommandResults {
		summary.Commands = append(summary.Commands, report.ReviewProbeCommandSummary{
			Command:         commandResult.Command,
			Args:            append([]string(nil), commandResult.Args...),
			WorkDir:         commandResult.WorkDir,
			Status:          commandResult.Status,
			ExitCode:        commandResult.ExitCode,
			OutputTruncated: commandResult.OutputTruncated,
			Error:           commandResult.Error,
			DurationMs:      commandResult.Duration.Milliseconds(),
		})
	}
	return summary
}

// BuildReviewProbeSummaries は複数 probe 結果をまとめて要約する。
func BuildReviewProbeSummaries(results []ReviewProbeResult) []report.ReviewProbeSummary {
	summaries := make([]report.ReviewProbeSummary, 0, len(results))
	for _, result := range results {
		summaries = append(summaries, BuildReviewProbeSummary(result))
	}
	return summaries
}
