package gathercontext

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

type prefetchPolicy struct {
	shouldPrefetch bool
	limit          int
	diagnostics    *search.SymbolBundleDiagnostics
	reason         string
	limited        bool
}

func prefetchPolicyForArtifact(metadata search.SearchExecutionMetadata) prefetchPolicy {
	limit, reason := prefetchLimitForDiagnostics(metadata.Diagnostics)
	policy := prefetchPolicy{
		limit:       limit,
		diagnostics: metadata.Diagnostics,
		limited:     limit < maxPrefetchedReads,
	}
	if !metadata.StructuredImpact {
		return policy
	}
	if metadata.Ambiguous {
		policy.reason = "ambiguous structured impact"
		return policy
	}
	if metadata.Bundle == nil || metadata.Bundle.Impact == nil || len(metadata.Bundle.Impact.RecommendedReads) == 0 {
		return policy
	}
	policy.shouldPrefetch = true
	policy.reason = reason
	return policy
}

func prefetchLimitForDiagnostics(diagnostics *search.SymbolBundleDiagnostics) (int, string) {
	limit := maxPrefetchedReads
	if diagnostics == nil {
		return limit, ""
	}

	var reasons []string
	if diagnosticFlagSet(diagnostics.Incomplete) {
		limit = min(limit, 1)
		reasons = append(reasons, "incomplete=true")
	}
	if diagnosticFlagSet(diagnostics.Truncated) {
		limit = min(limit, 1)
		reasons = append(reasons, "truncated=true")
	}
	if diagnosticFlagSet(diagnostics.BudgetLimitHit) {
		limit = min(limit, 1)
		reasons = append(reasons, "budget_limit_hit=true")
	}

	switch strings.TrimSpace(diagnostics.ResolvedBy) {
	case search.SymbolBundleResolvedByFallback, search.SymbolBundleResolvedByMixed:
		limit = min(limit, 1)
		reasons = append(reasons, "resolved_by="+strings.TrimSpace(diagnostics.ResolvedBy))
	}

	switch strings.TrimSpace(diagnostics.Confidence) {
	case search.SymbolBundleConfidenceLow:
		limit = min(limit, 1)
		reasons = append(reasons, "confidence="+search.SymbolBundleConfidenceLow)
	case search.SymbolBundleConfidenceMedium:
		limit = min(limit, 2)
		reasons = append(reasons, "confidence="+search.SymbolBundleConfidenceMedium)
	}

	if fallbackReason := strings.TrimSpace(diagnostics.FallbackReason); fallbackReason != "" && limit < maxPrefetchedReads {
		reasons = append(reasons, "fallback_reason="+fallbackReason)
	}

	return limit, strings.Join(reasons, ", ")
}

func diagnosticFlagSet(value *bool) bool {
	return value != nil && *value
}

func prefetchLimitedNote(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "Prefetch limited: diagnostics policy"
	}
	return "Prefetch limited: " + reason
}

func prefetchSkippedNote(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ""
	}
	return "Prefetch skipped: " + reason
}
