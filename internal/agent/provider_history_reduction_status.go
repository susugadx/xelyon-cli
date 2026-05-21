package agent

import (
	"fmt"
	"reflect"
)

func providerHistoryProjectionReportIsEmpty(report ProviderHistoryProjectionReport) bool {
	if len(report.Candidates) > 0 || len(report.Kept) > 0 {
		return false
	}
	report.Candidates = nil
	report.Kept = nil
	return reflect.DeepEqual(report, ProviderHistoryProjectionReport{})
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
	return fmt.Sprintf(
		"mode=%s; candidates=%s; replaced=%s; kept=%s; original=%s B; projected=%s B; saved=%s B",
		providerHistoryProjectionModeLabel(report.Mode),
		formatNumber(report.CandidateCount),
		formatNumber(report.ReplacedCount),
		formatNumber(report.KeptCount),
		formatNumber(report.OriginalBytes),
		formatNumber(report.ProjectedBytes),
		formatNumber(report.EstimatedSavedBytes),
	)
}

func providerHistoryReductionStatusSummary(runtime *AgentRuntime) (string, bool) {
	if runtime == nil {
		return "", false
	}
	report := runtime.LastProviderHistoryProjectionReport
	hasReport := !providerHistoryProjectionReportIsEmpty(report)
	resolution := providerHistoryReductionModeResolutionForRuntime(runtime)

	if resolution.configured == ProviderHistoryReductionAuto {
		prefix := fmt.Sprintf("mode=auto; effective=%s", providerHistoryProjectionModeLabel(resolution.effective))
		if hasReport {
			return fmt.Sprintf("%s; report: %s", prefix, formatProviderHistoryProjectionReportSummary(report)), true
		}
		return prefix + "; no report yet", true
	}

	if hasReport {
		return formatProviderHistoryProjectionReportSummary(report), true
	}
	if resolution.specified && resolution.configured != ProviderHistoryReductionDisabled {
		return fmt.Sprintf("mode=%s; no report yet", providerHistoryProjectionModeLabel(resolution.configured)), true
	}
	return "", false
}
