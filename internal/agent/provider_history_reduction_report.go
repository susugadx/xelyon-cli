package agent

import (
	"sort"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func buildProviderHistoryProjectionReport(original, projected []api.Message, policy ProviderHistoryReductionPolicy) ProviderHistoryProjectionReport {
	policy = normalizeProviderHistoryReductionPolicy(policy)
	if policy.Mode == ProviderHistoryReductionDisabled {
		return ProviderHistoryProjectionReport{}
	}

	report := buildProviderHistoryReductionDetectionReport(original, projected, policy.Mode)
	finalizeProviderHistoryProjectionReport(&report, original, projected)
	return report
}

func finalizeProviderHistoryProjectionReport(report *ProviderHistoryProjectionReport, original, projected []api.Message) {
	if report == nil || report.Mode == ProviderHistoryReductionDisabled {
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
	if report.OriginalBytes > report.ProjectedBytes {
		report.EstimatedSavedBytes = report.OriginalBytes - report.ProjectedBytes
	}
}

func countProviderHistoryReplacementApplied(candidates []ProviderHistoryReductionCandidate) int {
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
