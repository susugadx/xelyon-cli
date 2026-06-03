package agent

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

func providerHistoryProjectionReportIsEmpty(report ProviderHistoryProjectionReport) bool {
	if len(report.Candidates) > 0 || len(report.Kept) > 0 || len(report.KeptReasonCounts) > 0 {
		return false
	}
	if !providerHistoryCommandEditDryRunReportIsEmpty(report.CommandEditDryRun) {
		return false
	}
	report.Candidates = nil
	report.Kept = nil
	report.KeptReasonCounts = nil
	report.CommandEditDryRun = ProviderHistoryCommandEditDryRunReport{}
	return reflect.DeepEqual(report, ProviderHistoryProjectionReport{})
}

func providerHistoryCommandEditDryRunReportIsEmpty(report ProviderHistoryCommandEditDryRunReport) bool {
	if len(report.Candidates) > 0 || len(report.Kept) > 0 || len(report.CandidateReasonCounts) > 0 || len(report.KeptReasonCounts) > 0 {
		return false
	}
	report.Candidates = nil
	report.Kept = nil
	report.CandidateReasonCounts = nil
	report.KeptReasonCounts = nil
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
	return strings.Join([]string{
		modeLine,
		fmt.Sprintf("replacement_status=%s", providerHistoryReplacementStatusForStatus(report)),
		fmt.Sprintf(
			"content_replacements=%s; content_saved=%s B; approx_content_saved_tokens=%s",
			formatNumber(report.ReplacedCount),
			formatNumber(report.ContentReplacementSavedBytes),
			formatNumber(report.ApproxContentReplacementSavedTokens),
		),
		fmt.Sprintf(
			"command_output_replacements=%s; command_output_saved=%s B; approx_command_output_saved_tokens=%s",
			formatNumber(report.CommandEditDryRun.CommandReplacedCount),
			formatNumber(commandSavedBytes),
			formatNumber(commandSavedTokens),
		),
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
	}, "\n")
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
	return len(providerHistoryAppliedEvidencePointers(agent.Runtime.LastProviderHistoryProjectionReport))
}

func providerHistoryStatusSummaryLines(summary string) []string {
	if strings.TrimSpace(summary) == "" {
		return nil
	}
	return strings.Split(summary, "\n")
}
