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
	report.Candidates = nil
	report.Kept = nil
	report.KeptReasonCounts = nil
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
		"mode=%s; candidates=%s; replaced=%s; kept=%s; original=%s B; projected=%s B; saved=%s B; approx_saved_tokens=%s; kept_reasons=%s; responses_chain_disabled=%t",
		providerHistoryProjectionModeLabel(report.Mode),
		formatNumber(report.CandidateCount),
		formatNumber(report.ReplacedCount),
		formatNumber(report.KeptCount),
		formatNumber(report.OriginalBytes),
		formatNumber(report.ProjectedBytes),
		formatNumber(report.EstimatedSavedBytes),
		formatNumber(report.ApproxSavedTokens),
		formatProviderHistoryKeptReasonCounts(report.KeptReasonCounts),
		report.ResponsesChainDisabled,
	)
}

func formatProviderHistoryKeptReasonCounts(counts map[string]int) string {
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
