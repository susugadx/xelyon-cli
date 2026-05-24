package search

import (
	"context"
	"errors"
	"strings"
)

const (
	symbolBundleResolvedByLSP      = "lsp"
	symbolBundleResolvedByAST      = "ast"
	symbolBundleResolvedByFallback = "fallback"
	symbolBundleResolvedByMixed    = "mixed"
	symbolBundleResolvedByUnknown  = "unknown"

	symbolBundleConfidenceHigh    = "high"
	symbolBundleConfidenceMedium  = "medium"
	symbolBundleConfidenceLow     = "low"
	symbolBundleConfidenceUnknown = "unknown"

	symbolBundleFallbackReasonLSPUnavailable        = "lsp_unavailable"
	symbolBundleFallbackReasonLSPEmpty              = "lsp_empty"
	symbolBundleFallbackReasonLSPError              = "lsp_error"
	symbolBundleFallbackReasonLSPTimeout            = "lsp_timeout"
	symbolBundleFallbackReasonStructuredUnavailable = "structured_unavailable"
)

// Exported aliases expose the diagnostics value contract to orchestration packages.
const (
	SymbolBundleResolvedByLSP      = symbolBundleResolvedByLSP
	SymbolBundleResolvedByAST      = symbolBundleResolvedByAST
	SymbolBundleResolvedByFallback = symbolBundleResolvedByFallback
	SymbolBundleResolvedByMixed    = symbolBundleResolvedByMixed
	SymbolBundleResolvedByUnknown  = symbolBundleResolvedByUnknown

	SymbolBundleConfidenceHigh    = symbolBundleConfidenceHigh
	SymbolBundleConfidenceMedium  = symbolBundleConfidenceMedium
	SymbolBundleConfidenceLow     = symbolBundleConfidenceLow
	SymbolBundleConfidenceUnknown = symbolBundleConfidenceUnknown

	SymbolBundleFallbackReasonLSPUnavailable        = symbolBundleFallbackReasonLSPUnavailable
	SymbolBundleFallbackReasonLSPEmpty              = symbolBundleFallbackReasonLSPEmpty
	SymbolBundleFallbackReasonLSPError              = symbolBundleFallbackReasonLSPError
	SymbolBundleFallbackReasonLSPTimeout            = symbolBundleFallbackReasonLSPTimeout
	SymbolBundleFallbackReasonStructuredUnavailable = symbolBundleFallbackReasonStructuredUnavailable
)

func cloneSymbolBundleDiagnostics(diag SymbolBundleDiagnostics) SymbolBundleDiagnostics {
	cloned := diag
	cloned.LSPAttempted = cloneBoolPtr(diag.LSPAttempted)
	cloned.LSPAvailable = cloneBoolPtr(diag.LSPAvailable)
	cloned.LSPTimedOut = cloneBoolPtr(diag.LSPTimedOut)
	cloned.FallbackUsed = cloneBoolPtr(diag.FallbackUsed)
	cloned.Incomplete = cloneBoolPtr(diag.Incomplete)
	cloned.Truncated = cloneBoolPtr(diag.Truncated)
	cloned.BudgetLimitHit = cloneBoolPtr(diag.BudgetLimitHit)
	cloned.RawRefCount = cloneIntPtr(diag.RawRefCount)
	cloned.AcceptedRefCount = cloneIntPtr(diag.AcceptedRefCount)
	cloned.DroppedRefCount = cloneIntPtr(diag.DroppedRefCount)
	return cloned
}

func cloneBundleDiagnosticsForMetadata(bundle *SymbolBundle) *SymbolBundleDiagnostics {
	if bundle == nil {
		return nil
	}
	diagnostics := cloneSymbolBundleDiagnostics(bundle.Diagnostics)
	normalizeSymbolBundleDiagnostics(&diagnostics)
	return &diagnostics
}

func boolPtr(value bool) *bool {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func cloneBoolPtr(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func boolPtrValue(value *bool) bool {
	return value != nil && *value
}

func normalizeSymbolBundleDiagnostics(diag *SymbolBundleDiagnostics) {
	if diag == nil {
		return
	}
	diag.ResolvedBy = normalizeSymbolBundleResolvedBy(diag.ResolvedBy, diag.ResolvedViaLSP)
	if diag.Confidence == "" {
		diag.Confidence = symbolBundleConfidenceUnknown
	}
	if diag.Truncated != nil {
		diag.UpstreamTruncated = *diag.Truncated
	}
	if diag.Incomplete != nil {
		diag.UpstreamIncomplete = *diag.Incomplete
	}
	if diag.ResolvedBy == symbolBundleResolvedByLSP {
		diag.ResolvedViaLSP = true
	}
}

func normalizeSymbolBundleResolvedBy(resolvedBy string, legacyResolvedViaLSP bool) string {
	switch strings.TrimSpace(resolvedBy) {
	case symbolBundleResolvedByLSP:
		return symbolBundleResolvedByLSP
	case symbolBundleResolvedByAST:
		return symbolBundleResolvedByAST
	case symbolBundleResolvedByFallback:
		return symbolBundleResolvedByFallback
	case symbolBundleResolvedByMixed:
		return symbolBundleResolvedByMixed
	case symbolBundleResolvedByUnknown:
		return symbolBundleResolvedByUnknown
	}
	if legacyResolvedViaLSP {
		return symbolBundleResolvedByLSP
	}
	return symbolBundleResolvedByUnknown
}

func finalizeSymbolBundleDiagnostics(bundle *SymbolBundle) {
	if bundle == nil {
		return
	}
	if bundle.Diagnostics.BudgetLimitHit == nil {
		bundle.Diagnostics.BudgetLimitHit = boolPtr(symbolBundleHasBudgetLimitHit(bundle))
	} else if !*bundle.Diagnostics.BudgetLimitHit && symbolBundleHasBudgetLimitHit(bundle) {
		bundle.Diagnostics.BudgetLimitHit = boolPtr(true)
	}
	bundle.Diagnostics.Confidence = inferSymbolBundleConfidence(bundle.Diagnostics)
	normalizeSymbolBundleDiagnostics(&bundle.Diagnostics)
}

func symbolBundleHasBudgetLimitHit(bundle *SymbolBundle) bool {
	if bundle == nil {
		return false
	}
	for _, section := range bundle.Sections {
		if section.More {
			return true
		}
	}
	return false
}

func inferSymbolBundleConfidence(diag SymbolBundleDiagnostics) string {
	resolvedBy := normalizeSymbolBundleResolvedBy(diag.ResolvedBy, diag.ResolvedViaLSP)
	if boolPtrValue(diag.Incomplete) || boolPtrValue(diag.Truncated) || boolPtrValue(diag.BudgetLimitHit) {
		return symbolBundleConfidenceLow
	}
	switch resolvedBy {
	case symbolBundleResolvedByLSP:
		if diag.AcceptedRefCount != nil && *diag.AcceptedRefCount > 0 {
			return symbolBundleConfidenceHigh
		}
		return symbolBundleConfidenceMedium
	case symbolBundleResolvedByAST:
		return symbolBundleConfidenceMedium
	case symbolBundleResolvedByFallback, symbolBundleResolvedByMixed:
		return symbolBundleConfidenceLow
	default:
		return symbolBundleConfidenceUnknown
	}
}

func fallbackSearchDiagnostics(reason string) *SymbolBundleDiagnostics {
	if strings.TrimSpace(reason) == "" {
		reason = symbolBundleFallbackReasonStructuredUnavailable
	}
	diag := SymbolBundleDiagnostics{
		ResolvedBy:     symbolBundleResolvedByFallback,
		FallbackUsed:   boolPtr(true),
		FallbackReason: reason,
		Confidence:     symbolBundleConfidenceLow,
	}
	normalizeSymbolBundleDiagnostics(&diag)
	return &diag
}

func updateDiagnosticsRefCounts(diag *SymbolBundleDiagnostics, raw, accepted int) {
	if diag == nil {
		return
	}
	if raw >= 0 {
		diag.RawRefCount = intPtr(raw)
	}
	if accepted >= 0 {
		diag.AcceptedRefCount = intPtr(accepted)
	}
	if raw >= 0 && accepted >= 0 {
		dropped := raw - accepted
		if dropped < 0 {
			dropped = 0
		}
		diag.DroppedRefCount = intPtr(dropped)
	}
}

func lspReferenceErrorTimedOutForSearch(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "deadline") || strings.Contains(message, "timeout") || strings.Contains(message, "timed out")
}
