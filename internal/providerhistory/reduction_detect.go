package providerhistory

import "github.com/susugadx/xelyon-cli/internal/api"

func buildProviderHistoryReductionDetectionReport(original, projected []api.Message, policy Policy) ProjectionReport {
	policy = normalizePolicy(policy)
	mode := policy.Mode
	report := ProjectionReport{
		Mode:                  mode,
		OriginalMessageCount:  len(original),
		ProjectedMessageCount: len(projected),
		CommandEditDryRun:     newCommandEditDryRunReport(),
	}
	if len(original) == 0 {
		return report
	}

	assistantToolCallsByID := collectProviderHistoryAssistantToolCalls(original)
	trailingToolStart := providerHistoryTrailingToolSuffixStart(original)
	latestToolResultIndex := providerHistoryLatestToolResultIndex(original)
	report.CommandEditDryRun = buildCommandEditDryRunReport(original, projected, policy, assistantToolCallsByID, trailingToolStart, latestToolResultIndex)

	for i, msg := range original {
		if msg.Role != "tool" {
			continue
		}
		report.ToolResultCount++
		entry := providerHistoryReductionEntry(i, msg)

		linkage := resolveProviderHistoryToolResultLinkage(original, i, msg, assistantToolCallsByID, trailingToolStart, latestToolResultIndex)
		if linkage.ToolName != "" {
			entry.ToolName = linkage.ToolName
		}
		if linkage.KeepReason != "" {
			if recordProviderHistoryMCPRuntimePlaceholderCandidateFromContent(&report, policy, entry, msg.Content) {
				continue
			}
			entry.KeepReason = linkage.KeepReason
			report.Kept = append(report.Kept, entry)
			continue
		}
		toolName := linkage.ToolName
		entry.ToolName = toolName
		if toolName == "" {
			entry.KeepReason = "missing_tool_name"
			report.Kept = append(report.Kept, entry)
			continue
		}

		if isProviderHistoryReductionAlwaysKeptTool(toolName) {
			entry.KeepReason = "write_or_command_tool"
			report.Kept = append(report.Kept, entry)
			continue
		}
		if toolName == "run_skill_script" {
			recordProviderHistoryRunSkillScriptArtifactCandidate(&report, policy, entry, linkage.Ref.arguments, msg.Content, original)
			continue
		}
		if toolName == "web_search" {
			recordProviderHistoryWebSearchArtifactCandidate(&report, policy, entry, linkage.Ref.arguments, msg.Content, original)
			continue
		}
		if providerHistoryIsMCPToolResult(toolName) {
			if recordProviderHistoryMCPArtifactCandidate(&report, policy, entry, msg.Content) {
				continue
			}
		}
		if isProviderHistoryCandidateOnlyTool(toolName) {
			entry.CandidateOnly = true
			entry.FutureApplyCandidate = providerHistoryFutureApplyCandidate(toolName, msg.Content)
			entry.KeepReason = providerHistoryCandidateOnlyKeepReason(toolName, msg.Content)
			entry.Reason = "candidate_only_" + providerHistoryFutureFamilyName(toolName)
			entry.SuggestedReplacementKind = "candidate_only_keep"
			report.Candidates = append(report.Candidates, entry)
			report.Kept = append(report.Kept, entry)
			continue
		}
		if !isToolResultReductionCandidateTool(toolName) {
			entry.KeepReason = "tool_not_in_reduction_allowlist"
			report.Kept = append(report.Kept, entry)
			continue
		}
		if !providerHistoryHasLaterAssistant(original, i) {
			entry.KeepReason = "no_later_assistant_message"
			report.Kept = append(report.Kept, entry)
			continue
		}

		entry.Reason = providerHistoryReductionCandidateReason(toolName)
		entry.SuggestedReplacementKind = providerHistoryReductionReplacementKind(toolName)
		entry.SuggestedReplacementText = providerHistoryReductionSuggestedReplacementText(entry, linkage.Ref.arguments, msg.Content, original)
		report.Candidates = append(report.Candidates, entry)
		if mode == DryRun {
			kept := entry
			kept.KeepReason = "dry_run"
			report.Kept = append(report.Kept, kept)
		}
	}

	report.CandidateCount = len(report.Candidates)
	return report
}
