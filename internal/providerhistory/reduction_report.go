package providerhistory

import (
	"sort"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"github.com/susugadx/xelyon-cli/internal/token"
)

func buildProjectionReport(original, projected []api.Message, policy Policy) ProjectionReport {
	policy = normalizePolicy(policy)
	if policy.Mode == Disabled {
		return ProjectionReport{}
	}

	report := buildProviderHistoryReductionDetectionReport(original, projected, policy)
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
	report.ContentReplacementToolCounts = providerHistoryContentReplacementToolCounts(original, report.Candidates, report.Mode)
	report.SkillReplacementToolCounts = providerHistorySkillReplacementToolCounts(original, report.Candidates, report.Mode)
	report.FutureFamilyCandidateCounts = providerHistoryFutureFamilyCandidateCounts(report.Candidates)
	report.FutureFamilyKeptReasonCounts = providerHistoryFutureFamilyKeptReasonCounts(report.Kept)
	report.RawOutputRefs = appendProviderHistoryRawOutputRefs(report.RawOutputRefs, report.CommandEditDryRun.RawOutputRefs)
	report.RawOutputRefCount = len(report.RawOutputRefs)
	report.RawOutputArtifactCount = countProviderHistoryRawOutputArtifacts(report.RawOutputRefs)
	genericArtifactCandidates, genericEstimatedBytes, genericEstimatedTokens, genericActualBytes, genericActualTokens := providerHistoryGenericRawOutputArtifactMetrics(report.Candidates)
	report.DataBearingCandidateCount += report.CommandEditDryRun.ArtifactBackedCommandCandidates + genericArtifactCandidates
	report.ArtifactBackedEstimatedSavedBytes = report.CommandEditDryRun.ArtifactBackedCommandDryRunEstimatedSavedBytes + genericEstimatedBytes
	report.ApproxArtifactBackedEstimatedSavedTokens = report.CommandEditDryRun.ApproxArtifactBackedCommandDryRunEstimatedSavedTokens + genericEstimatedTokens
	report.ArtifactBackedActualSavedBytes = report.CommandEditDryRun.ArtifactBackedCommandReplacementSavedBytes + genericActualBytes
	report.ApproxArtifactBackedActualSavedTokens = report.CommandEditDryRun.ApproxArtifactBackedCommandReplacementSavedTokens + genericActualTokens
	report.EstimatedSavedBytes, report.ApproxSavedTokens = providerHistoryProviderFacingSavings(report)
	report.ReplacementStatus = providerHistoryProjectionReplacementStatus(report)
	report.KeptReasonCounts = countProviderHistoryKeptReasons(report.Kept)
	report.ResponsesChainDisabled = report.Mode == Apply && (report.ReplacedCount > 0 || report.CommandEditDryRun.CommandReplacedCount > 0 || report.CommandEditDryRun.EditArgReplacedCount > 0 || report.CommandEditDryRun.ArtifactBackedCommandReplacedCount > 0)
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
		if candidate.CandidateOnly {
			continue
		}
		if candidate.ArtifactBackedCandidate {
			continue
		}
		if mode == Apply && !candidate.ReplacementApplied {
			continue
		}
		if candidate.SuggestedReplacementText == "" || candidate.HistoryIndex < 0 || candidate.HistoryIndex >= len(original) {
			continue
		}
		originalContent := original[candidate.HistoryIndex].Content
		savedBytes, savedTokens, _, ok := providerHistoryContentReplacementEligibility(originalContent, candidate.SuggestedReplacementText)
		if !ok {
			continue
		}
		totalBytes += savedBytes
		totalTokens += savedTokens
	}
	return totalBytes, totalTokens
}

func providerHistoryContentReplacementToolCounts(original []api.Message, candidates []ReductionCandidate, mode Mode) map[string]int {
	if mode != DryRun && mode != Apply {
		return nil
	}
	counts := make(map[string]int)
	for _, candidate := range candidates {
		if candidate.CandidateOnly {
			continue
		}
		if candidate.ArtifactBackedCandidate {
			continue
		}
		if candidate.ToolName == "activate_skill" {
			continue
		}
		if candidate.ToolName == "" {
			continue
		}
		if mode == Apply && !candidate.ReplacementApplied {
			continue
		}
		if candidate.SuggestedReplacementText == "" || candidate.HistoryIndex < 0 || candidate.HistoryIndex >= len(original) {
			continue
		}
		if _, _, _, ok := providerHistoryContentReplacementEligibility(original[candidate.HistoryIndex].Content, candidate.SuggestedReplacementText); !ok {
			continue
		}
		counts[candidate.ToolName]++
	}
	if len(counts) == 0 {
		return nil
	}
	return counts
}

func providerHistorySkillReplacementToolCounts(original []api.Message, candidates []ReductionCandidate, mode Mode) map[string]int {
	if mode != DryRun && mode != Apply {
		return nil
	}
	counts := make(map[string]int)
	for _, candidate := range candidates {
		if candidate.CandidateOnly || candidate.ToolName != "activate_skill" {
			continue
		}
		if mode == Apply && !candidate.ReplacementApplied {
			continue
		}
		if candidate.SuggestedReplacementText == "" || candidate.HistoryIndex < 0 || candidate.HistoryIndex >= len(original) {
			continue
		}
		if _, _, _, ok := providerHistoryContentReplacementEligibility(original[candidate.HistoryIndex].Content, candidate.SuggestedReplacementText); !ok {
			continue
		}
		counts["activate_skill_duplicate"]++
	}
	if len(counts) == 0 {
		return nil
	}
	return counts
}

func providerHistoryFutureFamilyCandidateCounts(candidates []ReductionCandidate) map[string]int {
	counts := make(map[string]int)
	for _, candidate := range candidates {
		if !candidate.CandidateOnly {
			continue
		}
		family := providerHistoryFutureFamilyName(candidate.ToolName)
		if family == "" {
			continue
		}
		counts[family]++
	}
	if len(counts) == 0 {
		return nil
	}
	return counts
}

func providerHistoryFutureFamilyKeptReasonCounts(kept []ReductionCandidate) map[string]int {
	counts := make(map[string]int)
	for _, candidate := range kept {
		if !candidate.CandidateOnly || candidate.KeepReason == "" {
			continue
		}
		counts[candidate.KeepReason]++
	}
	if len(counts) == 0 {
		return nil
	}
	return counts
}

func providerHistoryProviderFacingSavings(report *ProjectionReport) (int, int) {
	if report == nil {
		return 0, 0
	}
	contentBytes := report.ContentReplacementSavedBytes
	contentTokens := report.ApproxContentReplacementSavedTokens
	commandBytes, commandTokens := providerHistoryCommandOutputSavings(report.CommandEditDryRun, report.Mode)
	editBytes, editTokens := providerHistoryEditArgSavings(report.CommandEditDryRun, report.Mode)
	artifactBytes, artifactTokens := providerHistoryArtifactBackedSavings(report, report.Mode)
	return contentBytes + commandBytes + editBytes + artifactBytes, contentTokens + commandTokens + editTokens + artifactTokens
}

func cloneProviderHistoryRawOutputRefs(refs []rawoutputs.RawOutputRef) []rawoutputs.RawOutputRef {
	if len(refs) == 0 {
		return nil
	}
	cloned := make([]rawoutputs.RawOutputRef, len(refs))
	copy(cloned, refs)
	return cloned
}

func appendProviderHistoryRawOutputRefs(first, second []rawoutputs.RawOutputRef) []rawoutputs.RawOutputRef {
	if len(first) == 0 {
		return cloneProviderHistoryRawOutputRefs(second)
	}
	out := cloneProviderHistoryRawOutputRefs(first)
	if len(second) > 0 {
		out = append(out, second...)
	}
	return out
}

func providerHistoryGenericRawOutputArtifactMetrics(candidates []ReductionCandidate) (candidatesCount, estimatedBytes, estimatedTokens, actualBytes, actualTokens int) {
	for _, candidate := range candidates {
		if !candidate.ArtifactBackedCandidate {
			continue
		}
		candidatesCount++
		estimatedBytes += candidate.EstimatedSavedBytes
		estimatedTokens += candidate.ApproxEstimatedSavedTokens
		if candidate.ReplacementApplied {
			actualBytes += candidate.ArtifactBackedActualSavedBytes
			actualTokens += candidate.ApproxArtifactBackedActualSavedTokens
		}
	}
	return candidatesCount, estimatedBytes, estimatedTokens, actualBytes, actualTokens
}

func countProviderHistoryRawOutputArtifacts(refs []rawoutputs.RawOutputRef) int {
	if len(refs) == 0 {
		return 0
	}
	artifactIDs := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if ref.ArtifactID == "" {
			continue
		}
		artifactIDs[ref.ArtifactID] = struct{}{}
	}
	return len(artifactIDs)
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

func providerHistoryArtifactBackedSavings(report *ProjectionReport, mode Mode) (int, int) {
	if mode == Apply {
		return report.ArtifactBackedActualSavedBytes, report.ApproxArtifactBackedActualSavedTokens
	}
	return 0, 0
}

func providerHistoryProjectionReplacementStatus(report *ProjectionReport) string {
	if report == nil || report.Mode != Apply {
		return providerHistoryReplacementStatusNotImplemented
	}
	actualReplacements := report.ReplacedCount + report.CommandEditDryRun.CommandReplacedCount + report.CommandEditDryRun.EditArgReplacedCount + report.CommandEditDryRun.ArtifactBackedCommandReplacedCount
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

func providerHistoryContentReplacementEligibility(originalContent, replacementText string) (int, int, string, bool) {
	if replacementText == "" {
		return 0, 0, "replacement_not_smaller", false
	}
	savedBytes := clampProviderHistorySavedBytes(len(originalContent), len(replacementText))
	if savedBytes == 0 {
		return 0, 0, "replacement_not_smaller", false
	}
	savedTokens := clampProviderHistorySavedTokens(
		token.EstimateTokenCount(originalContent),
		token.EstimateTokenCount(replacementText),
	)
	if savedTokens < providerHistoryContentReplacementMinSavedTokens {
		return 0, 0, "replacement_below_min_saved_tokens", false
	}
	return savedBytes, savedTokens, "", true
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
