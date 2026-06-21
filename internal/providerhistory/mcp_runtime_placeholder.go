package providerhistory

import (
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"strings"
)

const (
	mcpRuntimeCompactedResultKeepReason = "mcp_runtime_compacted_result_keep"
	rawOutputRefSourceMismatchReason    = "raw_output_ref_source_mismatch"
)

type mcpRuntimePlaceholder struct {
	rawOutputRefID string
	omittedReason  string
	surface        string
}

func recordProviderHistoryMCPRuntimePlaceholderCandidateFromContent(report *ProjectionReport, policy Policy, entry ReductionCandidate, content string) bool {
	if !providerHistoryIsMCPToolResult(entry.ToolName) {
		return false
	}
	placeholder, ok := parseMCPRuntimePlaceholder(content)
	if !ok {
		return false
	}
	recordProviderHistoryMCPRuntimePlaceholderCandidate(report, policy, entry, placeholder)
	return true
}

func recordProviderHistoryMCPRuntimePlaceholderCandidate(report *ProjectionReport, policy Policy, entry ReductionCandidate, placeholder mcpRuntimePlaceholder) {
	if report == nil {
		return
	}
	entry.CandidateOnly = false
	entry.FutureApplyCandidate = false
	entry.Reason = "runtime_compacted_mcp_tool_result"
	entry.SuggestedReplacementKind = "runtime_compacted_mcp_tool_result_keep"
	entry.KeepReason = mcpRuntimeCompactedResultKeepReason
	entry.RawOutputRefID = placeholder.rawOutputRefID
	if placeholder.rawOutputRefID == "" {
		if placeholder.omittedReason != "" {
			entry.KeepReason = placeholder.omittedReason
		} else {
			entry.KeepReason = "raw_output_ref_missing"
			entry.FailClosedReason = entry.KeepReason
		}
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return
	}
	activeContextAvailable := policy.RawOutputRehydrateContextEnabled && policy.ActiveContextTransportAvailable
	if activeContextAvailable {
		entry.RawOutputContextRequired = true
		if policy.RawOutputApplyDisabledReason != "" {
			entry.KeepReason = policy.RawOutputApplyDisabledReason
			entry.FailClosedReason = policy.RawOutputApplyDisabledReason
		}
	}
	ref, reason, ok := providerHistoryLookupRuntimeMCPRawOutputRef(policy, placeholder.rawOutputRefID)
	if !ok {
		entry.KeepReason = reason
		entry.FailClosedReason = reason
		report.RawOutputContextMissingRefIDs = append(report.RawOutputContextMissingRefIDs, placeholder.rawOutputRefID)
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return
	}
	if ref.Surface != string(rawoutputs.SurfaceMCPToolResult) {
		reason := "raw_output_ref_surface_mismatch"
		entry.KeepReason = reason
		entry.FailClosedReason = reason
		report.RawOutputContextMissingRefIDs = append(report.RawOutputContextMissingRefIDs, placeholder.rawOutputRefID)
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return
	}
	if !providerHistoryRuntimeMCPRawOutputRefMatchesEntry(ref, entry) {
		entry.KeepReason = rawOutputRefSourceMismatchReason
		entry.FailClosedReason = rawOutputRefSourceMismatchReason
		report.RawOutputContextMissingRefIDs = append(report.RawOutputContextMissingRefIDs, placeholder.rawOutputRefID)
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return
	}
	report.RawOutputContextRefs = append(report.RawOutputContextRefs, ref)
	if !activeContextAvailable {
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return
	}
	report.Candidates = append(report.Candidates, entry)
	report.Kept = append(report.Kept, entry)
}

func parseMCPRuntimePlaceholder(content string) (mcpRuntimePlaceholder, bool) {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "[compacted MCP tool result;") {
		return mcpRuntimePlaceholder{}, false
	}
	header := trimmed
	if end := strings.Index(header, "]"); end >= 0 {
		header = header[:end]
	}
	placeholder := mcpRuntimePlaceholder{}
	fields := strings.Split(strings.ReplaceAll(header, "\n", ";"), ";")
	for _, field := range fields {
		field = strings.TrimSpace(field)
		switch {
		case strings.HasPrefix(field, "surface="):
			placeholder.surface = strings.TrimSpace(strings.TrimPrefix(field, "surface="))
		case strings.HasPrefix(field, "raw_output_ref="):
			placeholder.rawOutputRefID = strings.TrimSpace(strings.TrimPrefix(field, "raw_output_ref="))
		case strings.HasPrefix(field, "full_output_omitted_reason="):
			placeholder.omittedReason = strings.TrimSpace(strings.TrimPrefix(field, "full_output_omitted_reason="))
		}
	}
	if placeholder.surface != string(rawoutputs.SurfaceMCPToolResult) {
		return mcpRuntimePlaceholder{}, false
	}
	if placeholder.rawOutputRefID == "" && placeholder.omittedReason == "" {
		return mcpRuntimePlaceholder{}, false
	}
	return placeholder, true
}
