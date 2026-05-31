package providerhistory

import (
	"sort"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/token"
)

func buildProjectionReport(original, projected []api.Message, policy Policy) ProjectionReport {
	policy = normalizePolicy(policy)
	if policy.Mode == Disabled {
		return ProjectionReport{}
	}

	report := buildProviderHistoryReductionDetectionReport(original, projected, policy.Mode)
	finalizeProjectionReport(&report, original, projected)
	return report
}

func finalizeProjectionReport(report *ProjectionReport, original, projected []api.Message) {
	if report == nil || report.Mode == Disabled {
		return
	}
	report.CandidateCount = len(report.Candidates)
	report.ReplacedCount = countProviderHistoryReplacementApplied(report.Candidates)
	report.KeptCount = report.ToolResultCount - report.ReplacedCount
	if report.KeptCount < 0 {
		report.KeptCount = 0
	}
	sort.SliceStable(report.Kept, func(i, j int) bool {
		return report.Kept[i].HistoryIndex < report.Kept[j].HistoryIndex
	})
	report.OriginalBytes = providerHistoryContentBytes(original)
	report.ProjectedBytes = providerHistoryContentBytes(projected)
	report.EstimatedSavedBytes = 0
	if report.OriginalBytes > report.ProjectedBytes {
		report.EstimatedSavedBytes = report.OriginalBytes - report.ProjectedBytes
	}
	report.ApproxSavedTokens = providerHistoryApproxSavedTokens(original, projected)
	report.KeptReasonCounts = countProviderHistoryKeptReasons(report.Kept)
	report.ResponsesChainDisabled = report.Mode == Apply && (report.ReplacedCount > 0 || report.CommandEditDryRun.CommandReplacedCount > 0 || report.CommandEditDryRun.EditArgReplacedCount > 0)
}

func countProviderHistoryReplacementApplied(candidates []ReductionCandidate) int {
	count := 0
	for _, candidate := range candidates {
		if candidate.ReplacementApplied {
			count++
		}
	}
	return count
}

func providerHistoryContentBytes(messages []api.Message) int {
	total := 0
	for _, msg := range messages {
		total += len(msg.Content)
	}
	return total
}

func providerHistoryApproxSavedTokens(original, projected []api.Message) int {
	originalTokens := providerHistoryContentTokens(original)
	projectedTokens := providerHistoryContentTokens(projected)
	if originalTokens <= projectedTokens {
		return 0
	}
	return originalTokens - projectedTokens
}

func providerHistoryContentTokens(messages []api.Message) int {
	total := 0
	for _, msg := range messages {
		total += token.EstimateTokenCount(msg.Content)
	}
	return total
}

func countProviderHistoryKeptReasons(kept []ReductionCandidate) map[string]int {
	if len(kept) == 0 {
		return nil
	}
	counts := make(map[string]int)
	for _, candidate := range kept {
		if candidate.KeepReason == "" {
			continue
		}
		counts[candidate.KeepReason]++
	}
	if len(counts) == 0 {
		return nil
	}
	return counts
}
