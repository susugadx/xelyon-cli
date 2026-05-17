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
		return "disabled"
	case ProviderHistoryReductionDryRun:
		return "dry-run"
	case ProviderHistoryReductionApply:
		return "apply"
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
	if !providerHistoryProjectionReportIsEmpty(report) {
		return formatProviderHistoryProjectionReportSummary(report), true
	}
	if runtime.Options.EnableProviderHistoryReduction {
		return "enabled; no report yet", true
	}
	return "", false
}
