package agent

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/providerhistory"
	"github.com/susugadx/xelyon-cli/internal/review"
)

func providerHistoryProjectionReportIsEmpty(report ProviderHistoryProjectionReport) bool {
	if len(report.Candidates) > 0 ||
		len(report.Kept) > 0 ||
		len(report.KeptReasonCounts) > 0 ||
		len(report.ContentReplacementToolCounts) > 0 ||
		len(report.SkillReplacementToolCounts) > 0 ||
		len(report.FutureFamilyCandidateCounts) > 0 ||
		len(report.FutureFamilyKeptReasonCounts) > 0 ||
		len(report.RawOutputRefs) > 0 ||
		report.RawOutputRefCount > 0 ||
		report.RawOutputArtifactCount > 0 ||
		report.DataBearingCandidateCount > 0 ||
		report.ArtifactBackedEstimatedSavedBytes > 0 ||
		report.ApproxArtifactBackedEstimatedSavedTokens > 0 ||
		report.ArtifactBackedActualSavedBytes > 0 ||
		report.ApproxArtifactBackedActualSavedTokens > 0 {
		return false
	}
	if !providerHistoryCommandEditDryRunReportIsEmpty(report.CommandEditDryRun) {
		return false
	}
	report.Candidates = nil
	report.Kept = nil
	report.KeptReasonCounts = nil
	report.ContentReplacementToolCounts = nil
	report.SkillReplacementToolCounts = nil
	report.FutureFamilyCandidateCounts = nil
	report.FutureFamilyKeptReasonCounts = nil
	report.RawOutputRefs = nil
	report.CommandEditDryRun = ProviderHistoryCommandEditDryRunReport{}
	return reflect.DeepEqual(report, ProviderHistoryProjectionReport{})
}

func providerHistoryCommandEditDryRunReportIsEmpty(report ProviderHistoryCommandEditDryRunReport) bool {
	if len(report.Candidates) > 0 ||
		len(report.Kept) > 0 ||
		len(report.CandidateReasonCounts) > 0 ||
		len(report.CommandReplacementClassifierCounts) > 0 ||
		len(report.KeptReasonCounts) > 0 ||
		len(report.RawOutputRefs) > 0 ||
		report.ArtifactBackedCommandCandidates > 0 ||
		report.ArtifactBackedCommandApplyEligible > 0 ||
		report.ArtifactBackedCommandReplacedCount > 0 ||
		report.ArtifactBackedCommandDryRunEstimatedSavedBytes > 0 ||
		report.ApproxArtifactBackedCommandDryRunEstimatedSavedTokens > 0 ||
		report.ArtifactBackedCommandReplacementSavedBytes > 0 ||
		report.ApproxArtifactBackedCommandReplacementSavedTokens > 0 ||
		len(report.ArtifactBackedKeptReasonCounts) > 0 {
		return false
	}
	report.Candidates = nil
	report.Kept = nil
	report.CandidateReasonCounts = nil
	report.CommandReplacementClassifierCounts = nil
	report.KeptReasonCounts = nil
	report.RawOutputRefs = nil
	report.ArtifactBackedKeptReasonCounts = nil
	if report.ReplacementStatus == providerHistoryCommandEditReplacementStatusNotImplemented {
		report.ReplacementStatus = ""
	}
	return reflect.DeepEqual(report, ProviderHistoryCommandEditDryRunReport{})
}

func providerHistoryProjectionModeLabel(mode ProviderHistoryReductionMode) string {
	switch mode {
	case ProviderHistoryReductionDisabled:
		return "off"
	case ProviderHistoryReductionDryRun:
		return "dry_run"
	case ProviderHistoryReductionApply:
		return "apply"
	case ProviderHistoryReductionAuto:
		return "auto"
	default:
		return "unknown"
	}
}

func formatProviderHistoryProjectionReportSummary(report ProviderHistoryProjectionReport) string {
	return formatProviderHistoryProjectionReportSummaryWithModeLine(
		report,
		fmt.Sprintf("provider history reduction: %s", providerHistoryProjectionModeLabel(report.Mode)),
	)
}

func formatProviderHistoryProjectionReportSummaryWithModeLine(report ProviderHistoryProjectionReport, modeLine string) string {
	commandSavedBytes, commandSavedTokens := providerHistoryCommandOutputSavingsForStatus(report)
	editSavedBytes, editSavedTokens := providerHistoryEditArgSavingsForStatus(report)
	lines := []string{
		modeLine,
		fmt.Sprintf("replacement_status=%s", providerHistoryReplacementStatusForStatus(report)),
		fmt.Sprintf(
			"content_replacements=%s; content_saved=%s B; approx_content_saved_tokens=%s",
			formatNumber(report.ReplacedCount),
			formatNumber(report.ContentReplacementSavedBytes),
			formatNumber(report.ApproxContentReplacementSavedTokens),
		),
	}
	if len(report.ContentReplacementToolCounts) > 0 {
		lines = append(lines, fmt.Sprintf("content_replacement_tools=%s", formatProviderHistoryCountMap(report.ContentReplacementToolCounts)))
	}
	if len(report.SkillReplacementToolCounts) > 0 {
		lines = append(lines, fmt.Sprintf("skill_replacement_tools=%s", formatProviderHistoryCountMap(report.SkillReplacementToolCounts)))
	}
	lines = append(lines,
		fmt.Sprintf(
			"command_output_replacements=%s; command_output_saved=%s B; approx_command_output_saved_tokens=%s",
			formatNumber(report.CommandEditDryRun.CommandReplacedCount),
			formatNumber(commandSavedBytes),
			formatNumber(commandSavedTokens),
		),
	)
	if len(report.CommandEditDryRun.CommandReplacementClassifierCounts) > 0 {
		lines = append(lines, fmt.Sprintf("command_output_tools=%s", formatProviderHistoryCountMap(report.CommandEditDryRun.CommandReplacementClassifierCounts)))
	}
	if providerHistoryReportHasRawOutputArtifactSummary(report) {
		lines = append(lines,
			fmt.Sprintf(
				"raw_output_refs=%s; raw_output_artifacts=%s; data_bearing_candidates=%s",
				formatNumber(report.RawOutputRefCount),
				formatNumber(report.RawOutputArtifactCount),
				formatNumber(report.DataBearingCandidateCount),
			),
			fmt.Sprintf(
				"artifact_backed_command_candidates=%s; artifact_backed_command_apply_eligible=%s; artifact_backed_command_replacements=%s",
				formatNumber(report.CommandEditDryRun.ArtifactBackedCommandCandidates),
				formatNumber(report.CommandEditDryRun.ArtifactBackedCommandApplyEligible),
				formatNumber(report.CommandEditDryRun.ArtifactBackedCommandReplacedCount),
			),
			fmt.Sprintf(
				"artifact_backed_estimated_saved=%s B; approx_artifact_backed_estimated_saved_tokens=%s",
				formatNumber(report.ArtifactBackedEstimatedSavedBytes),
				formatNumber(report.ApproxArtifactBackedEstimatedSavedTokens),
			),
			fmt.Sprintf(
				"artifact_backed_saved=%s B; approx_artifact_backed_saved_tokens=%s",
				formatNumber(report.ArtifactBackedActualSavedBytes),
				formatNumber(report.ApproxArtifactBackedActualSavedTokens),
			),
		)
		if len(report.CommandEditDryRun.ArtifactBackedKeptReasonCounts) > 0 {
			lines = append(lines, fmt.Sprintf("artifact_backed_command_kept_reasons=%s", formatProviderHistoryReasonCounts(report.CommandEditDryRun.ArtifactBackedKeptReasonCounts)))
		}
	}
	lines = append(lines,
		fmt.Sprintf(
			"edit_arg_replacements=%s; edit_arg_saved=%s B; approx_edit_arg_saved_tokens=%s",
			formatNumber(report.CommandEditDryRun.EditArgReplacedCount),
			formatNumber(editSavedBytes),
			formatNumber(editSavedTokens),
		),
		fmt.Sprintf(
			"total_provider_facing_saved=%s B; approx_total_provider_facing_saved_tokens=%s",
			formatNumber(report.EstimatedSavedBytes),
			formatNumber(report.ApproxSavedTokens),
		),
		fmt.Sprintf("responses_chain_disabled=%t", report.ResponsesChainDisabled),
	)
	if len(report.FutureFamilyCandidateCounts) > 0 {
		lines = append(lines, fmt.Sprintf("future_family_candidates=%s", formatProviderHistoryCountMap(report.FutureFamilyCandidateCounts)))
	}
	if len(report.FutureFamilyCandidateCounts) > 0 && len(report.FutureFamilyKeptReasonCounts) > 0 {
		lines = append(lines, fmt.Sprintf("future_family_kept_reasons=%s", formatProviderHistoryReasonCounts(report.FutureFamilyKeptReasonCounts)))
	}
	return strings.Join(lines, "\n")
}

func providerHistoryReportHasRawOutputArtifactSummary(report ProviderHistoryProjectionReport) bool {
	return report.RawOutputRefCount > 0 ||
		report.RawOutputArtifactCount > 0 ||
		report.DataBearingCandidateCount > 0 ||
		report.ArtifactBackedEstimatedSavedBytes > 0 ||
		report.ApproxArtifactBackedEstimatedSavedTokens > 0 ||
		report.ArtifactBackedActualSavedBytes > 0 ||
		report.ApproxArtifactBackedActualSavedTokens > 0 ||
		report.CommandEditDryRun.ArtifactBackedCommandCandidates > 0 ||
		report.CommandEditDryRun.ArtifactBackedCommandApplyEligible > 0 ||
		report.CommandEditDryRun.ArtifactBackedCommandReplacedCount > 0 ||
		len(report.CommandEditDryRun.ArtifactBackedKeptReasonCounts) > 0
}

func providerHistoryReplacementStatusForStatus(report ProviderHistoryProjectionReport) string {
	if strings.TrimSpace(report.ReplacementStatus) != "" {
		return report.ReplacementStatus
	}
	return providerHistoryCommandEditReplacementStatusNotImplemented
}

func providerHistoryCommandOutputSavingsForStatus(report ProviderHistoryProjectionReport) (int, int) {
	if report.Mode == ProviderHistoryReductionApply {
		return report.CommandEditDryRun.CommandReplacementSavedBytes, report.CommandEditDryRun.ApproxCommandReplacementSavedTokens
	}
	if report.Mode == ProviderHistoryReductionDryRun {
		return report.CommandEditDryRun.CommandEstimatedSavedBytes, report.CommandEditDryRun.ApproxCommandSavedTokens
	}
	return 0, 0
}

func providerHistoryEditArgSavingsForStatus(report ProviderHistoryProjectionReport) (int, int) {
	if report.Mode == ProviderHistoryReductionApply {
		return report.CommandEditDryRun.EditArgReplacementSavedBytes, report.CommandEditDryRun.ApproxEditArgReplacementSavedTokens
	}
	if report.Mode == ProviderHistoryReductionDryRun {
		return report.CommandEditDryRun.EditArgEstimatedSavedBytes, report.CommandEditDryRun.ApproxEditArgSavedTokens
	}
	return 0, 0
}

func formatProviderHistoryReasonCounts(counts map[string]int) string {
	return formatProviderHistoryCountMap(counts)
}

func formatProviderHistoryCountMap(counts map[string]int) string {
	if len(counts) == 0 {
		return "none"
	}
	reasons := make([]string, 0, len(counts))
	for reason := range counts {
		if strings.TrimSpace(reason) == "" {
			continue
		}
		reasons = append(reasons, reason)
	}
	if len(reasons) == 0 {
		return "none"
	}
	sort.Strings(reasons)
	parts := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		parts = append(parts, fmt.Sprintf("%s:%d", reason, counts[reason]))
	}
	return strings.Join(parts, ", ")
}

func providerHistoryReductionStatusSummary(runtime *AgentRuntime) (string, bool) {
	if runtime == nil {
		return "", false
	}
	report := runtime.LastProviderHistoryProjectionReport
	hasReport := !providerHistoryProjectionReportIsEmpty(report)
	resolution := providerHistoryReductionModeResolutionForRuntime(runtime)

	if resolution.configured == ProviderHistoryReductionAuto {
		if hasReport {
			modeLine := fmt.Sprintf("provider history reduction: auto; effective=%s", providerHistoryProjectionModeLabel(resolution.effective))
			return formatProviderHistoryProjectionReportSummaryWithModeLine(report, modeLine), true
		}
		return fmt.Sprintf("provider history reduction: auto; effective=%s; no report yet", providerHistoryProjectionModeLabel(resolution.effective)), true
	}

	if hasReport {
		return formatProviderHistoryProjectionReportSummary(report), true
	}
	if resolution.specified && resolution.configured != ProviderHistoryReductionDisabled {
		return fmt.Sprintf("provider history reduction: %s; no report yet", providerHistoryProjectionModeLabel(resolution.configured)), true
	}
	return "", false
}

func providerHistoryRehydrateContextStatusSummary(agent *Agent) string {
	transport := providerHistoryActiveContextTransportForStatus(agent)
	evidenceCount := providerHistoryRehydratedEvidenceCountForStatus(agent)
	return fmt.Sprintf(
		"rehydrate_context=%s; active_context_transport=%s; active_context_rehydrated_evidence=%t; count=%s",
		onOffProviderHistoryRehydrateContext(providerHistoryRehydrateContextEnabled(agent)),
		transport,
		providerHistoryRehydrateContextEnabled(agent) && transport != "none" && evidenceCount > 0,
		formatNumber(evidenceCount),
	)
}

func reviewPromptReductionStatusSummary(runtime *AgentRuntime) (string, bool) {
	if runtime == nil {
		return "", false
	}
	report := review.CloneReviewPromptReductionReport(runtime.LastReviewPromptReductionReport)
	if review.ReviewPromptReductionReportIsEmpty(report) {
		return "", false
	}

	mode := reviewPromptReductionModeLabel(report.Mode)
	replacementSavedBytes, replacementSavedTokens := reviewPromptReductionSavingsForStatus(report)
	lines := []string{
		fmt.Sprintf("review prompt reduction: %s", mode),
		fmt.Sprintf(
			"review_history_candidates=%s; review_history_replacements=%s",
			formatNumber(report.CandidateCount),
			formatNumber(report.ReplacedCount),
		),
		fmt.Sprintf(
			"review_state_summaries=%s; absorbed_intermediate=%s; quality_floor=%s",
			formatNumber(report.StateSummaryCount),
			formatNumber(report.AbsorbedItemCount),
			reviewPromptReductionQualityFloorLabel(report.QualityFloorPreserved),
		),
		fmt.Sprintf(
			"review_history_estimated_saved=%s B; approx_review_history_estimated_saved_tokens=%s",
			formatNumber(report.EstimatedSavedBytes),
			formatNumber(report.ApproxEstimatedSavedTokens),
		),
		fmt.Sprintf(
			"review_history_saved=%s B; approx_review_history_saved_tokens=%s",
			formatNumber(replacementSavedBytes),
			formatNumber(replacementSavedTokens),
		),
	}
	if report.RawOutputLedgerCount > 0 ||
		report.RawOutputRequiredRefCount > 0 ||
		report.RawOutputRehydratedRefCount > 0 ||
		report.RawOutputMissingRefCount > 0 ||
		report.RawOutputBudgetExhaustedRefCount > 0 {
		lines = append(lines, fmt.Sprintf(
			"review_raw_output_ledgers=%s; required_refs=%s; rehydrated_refs=%s; missing_refs=%s; budget_exhausted_refs=%s",
			formatNumber(report.RawOutputLedgerCount),
			formatNumber(report.RawOutputRequiredRefCount),
			formatNumber(report.RawOutputRehydratedRefCount),
			formatNumber(report.RawOutputMissingRefCount),
			formatNumber(report.RawOutputBudgetExhaustedRefCount),
		))
	}
	if len(report.ClassifierCounts) > 0 {
		lines = append(lines, fmt.Sprintf("review_history_tools=%s", formatProviderHistoryCountMap(report.ClassifierCounts)))
	}
	if len(report.FamilyCounts) > 0 {
		lines = append(lines, fmt.Sprintf("review_history_families=%s", formatProviderHistoryCountMap(report.FamilyCounts)))
	}
	if len(report.StatusCounts) > 0 {
		lines = append(lines, fmt.Sprintf("review_history_statuses=%s", formatProviderHistoryCountMap(report.StatusCounts)))
	}
	if len(report.KeptReasonCounts) > 0 {
		lines = append(lines, fmt.Sprintf("review_history_kept_reasons=%s", formatProviderHistoryCountMap(report.KeptReasonCounts)))
	}
	return strings.Join(lines, "\n"), true
}

func reviewPromptReductionModeLabel(mode review.ReviewPromptReductionMode) string {
	switch mode {
	case review.ReviewPromptReductionModeApply:
		return "apply"
	case review.ReviewPromptReductionModeDryRun:
		return "dry_run"
	default:
		return "off"
	}
}

func reviewPromptReductionSavingsForStatus(report review.ReviewPromptReductionReport) (int, int) {
	if report.Mode != review.ReviewPromptReductionModeApply {
		return 0, 0
	}
	return report.ReplacementSavedBytes, report.ApproxReplacementSavedTokens
}

func reviewPromptReductionQualityFloorLabel(preserved bool) string {
	if preserved {
		return "preserved"
	}
	return "unknown"
}

func providerHistoryRehydrateContextEnabled(agent *Agent) bool {
	return agent != nil &&
		agent.Runtime != nil &&
		agent.Runtime.Options.EnableProviderHistoryRehydrateContext
}

func providerHistoryActiveContextTransportForStatus(agent *Agent) string {
	if agent == nil {
		return "none"
	}
	return string(agent.providerActiveContextTransport())
}

func onOffProviderHistoryRehydrateContext(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}

func providerHistoryRehydratedEvidenceCountForStatus(agent *Agent) int {
	if agent == nil || agent.Runtime == nil {
		return 0
	}
	return len(providerhistory.AppliedEvidencePointers(agent.Runtime.LastProviderHistoryProjectionReport))
}

func providerHistoryStatusSummaryLines(summary string) []string {
	if strings.TrimSpace(summary) == "" {
		return nil
	}
	return strings.Split(summary, "\n")
}
