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
	report.ContentReplacementSavedBytes, report.ApproxContentReplacementSavedTokens = providerHistoryContentReplacementSavings(original, report.Candidates, report.Mode)
	report.EstimatedSavedBytes, report.ApproxSavedTokens = providerHistoryProviderFacingSavings(report)
	report.ReplacementStatus = providerHistoryProjectionReplacementStatus(report)
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

func providerHistoryContentReplacementSavings(original []api.Message, candidates []ReductionCandidate, mode Mode) (int, int) {
	if mode != DryRun && mode != Apply {
		return 0, 0
	}
	totalBytes := 0
	totalTokens := 0
	for _, candidate := range candidates {
		if mode == Apply && !candidate.ReplacementApplied {
			continue
		}
		if candidate.SuggestedReplacementText == "" || candidate.HistoryIndex < 0 || candidate.HistoryIndex >= len(original) {
			continue
		}
		originalContent := original[candidate.HistoryIndex].Content
		savedBytes := clampProviderHistorySavedBytes(len(originalContent), len(candidate.SuggestedReplacementText))
		if savedBytes == 0 {
			continue
		}
		totalBytes += savedBytes
		totalTokens += clampProviderHistorySavedTokens(
			token.EstimateTokenCount(originalContent),
			token.EstimateTokenCount(candidate.SuggestedReplacementText),
		)
	}
	return totalBytes, totalTokens
}

func providerHistoryProviderFacingSavings(report *ProjectionReport) (int, int) {
	if report == nil {
		return 0, 0
	}
	contentBytes := report.ContentReplacementSavedBytes
	contentTokens := report.ApproxContentReplacementSavedTokens
	commandBytes, commandTokens := providerHistoryCommandOutputSavings(report.CommandEditDryRun, report.Mode)
	editBytes, editTokens := providerHistoryEditArgSavings(report.CommandEditDryRun, report.Mode)
	return contentBytes + commandBytes + editBytes, contentTokens + commandTokens + editTokens
}

func providerHistoryCommandOutputSavings(report CommandEditDryRunReport, mode Mode) (int, int) {
	if mode == Apply {
		return report.CommandReplacementSavedBytes, report.ApproxCommandReplacementSavedTokens
	}
	if mode == DryRun {
		return report.CommandEstimatedSavedBytes, report.ApproxCommandSavedTokens
	}
	return 0, 0
}

func providerHistoryEditArgSavings(report CommandEditDryRunReport, mode Mode) (int, int) {
	if mode == Apply {
		return report.EditArgReplacementSavedBytes, report.ApproxEditArgReplacementSavedTokens
	}
	if mode == DryRun {
		return report.EditArgEstimatedSavedBytes, report.ApproxEditArgSavedTokens
	}
	return 0, 0
}

func providerHistoryProjectionReplacementStatus(report *ProjectionReport) string {
	if report == nil || report.Mode != Apply {
		return providerHistoryReplacementStatusNotImplemented
	}
	actualReplacements := report.ReplacedCount + report.CommandEditDryRun.CommandReplacedCount + report.CommandEditDryRun.EditArgReplacedCount
	if actualReplacements == 0 {
		return providerHistoryReplacementStatusNotImplemented
	}
	detectedCandidates := report.CandidateCount + report.CommandEditDryRun.CommandCandidates + report.CommandEditDryRun.EditArgCandidates
	if actualReplacements == detectedCandidates {
		return providerHistoryReplacementStatusApply
	}
	return providerHistoryReplacementStatusPartialApply
}

func clampProviderHistorySavedBytes(originalBytes, replacementBytes int) int {
	if originalBytes <= replacementBytes {
		return 0
	}
	return originalBytes - replacementBytes
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
