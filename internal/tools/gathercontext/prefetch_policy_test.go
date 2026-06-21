package gathercontext

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

func TestPrefetchPolicyForArtifactLimitsFromDiagnostics(t *testing.T) {
	tests := []struct {
		name           string
		diagnostics    *search.SymbolBundleDiagnostics
		wantLimit      int
		wantLimited    bool
		reasonContains []string
	}{
		{
			name: "high LSP keeps three reads",
			diagnostics: &search.SymbolBundleDiagnostics{
				ResolvedBy: search.SymbolBundleResolvedByLSP,
				Confidence: search.SymbolBundleConfidenceHigh,
			},
			wantLimit: 3,
		},
		{
			name: "medium AST limits to two reads",
			diagnostics: &search.SymbolBundleDiagnostics{
				ResolvedBy: search.SymbolBundleResolvedByAST,
				Confidence: search.SymbolBundleConfidenceMedium,
			},
			wantLimit:      2,
			wantLimited:    true,
			reasonContains: []string{"confidence=medium"},
		},
		{
			name: "low fallback limits to one read",
			diagnostics: &search.SymbolBundleDiagnostics{
				ResolvedBy: search.SymbolBundleResolvedByFallback,
				Confidence: search.SymbolBundleConfidenceLow,
			},
			wantLimit:      1,
			wantLimited:    true,
			reasonContains: []string{"resolved_by=fallback", "confidence=low"},
		},
		{
			name: "mixed LSP error limits to one read and carries fallback reason",
			diagnostics: &search.SymbolBundleDiagnostics{
				ResolvedBy:     search.SymbolBundleResolvedByMixed,
				Confidence:     search.SymbolBundleConfidenceLow,
				FallbackReason: search.SymbolBundleFallbackReasonLSPError,
			},
			wantLimit:      1,
			wantLimited:    true,
			reasonContains: []string{"resolved_by=mixed", "fallback_reason=lsp_error"},
		},
		{
			name: "truncated diagnostics limits to one read",
			diagnostics: &search.SymbolBundleDiagnostics{
				ResolvedBy: search.SymbolBundleResolvedByLSP,
				Confidence: search.SymbolBundleConfidenceHigh,
				Truncated:  prefetchBoolPtr(true),
			},
			wantLimit:      1,
			wantLimited:    true,
			reasonContains: []string{"truncated=true"},
		},
		{
			name: "budget limit hit diagnostics limits to one read",
			diagnostics: &search.SymbolBundleDiagnostics{
				ResolvedBy:     search.SymbolBundleResolvedByAST,
				Confidence:     search.SymbolBundleConfidenceMedium,
				BudgetLimitHit: prefetchBoolPtr(true),
			},
			wantLimit:      1,
			wantLimited:    true,
			reasonContains: []string{"budget_limit_hit=true"},
		},
		{
			name: "incomplete diagnostics limits to one read",
			diagnostics: &search.SymbolBundleDiagnostics{
				ResolvedBy:  search.SymbolBundleResolvedByAST,
				Confidence:  search.SymbolBundleConfidenceMedium,
				Incomplete:  prefetchBoolPtr(true),
				RawRefCount: nil,
			},
			wantLimit:      1,
			wantLimited:    true,
			reasonContains: []string{"incomplete=true"},
		},
		{
			name:      "nil diagnostics keep compatibility limit of three reads",
			wantLimit: 3,
		},
		{
			name: "unknown confidence keeps compatibility limit of three reads",
			diagnostics: &search.SymbolBundleDiagnostics{
				ResolvedBy: search.SymbolBundleResolvedByUnknown,
				Confidence: search.SymbolBundleConfidenceUnknown,
			},
			wantLimit: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := prefetchPolicyForArtifact(prefetchPolicyMetadata(tt.diagnostics))
			if !policy.shouldPrefetch {
				t.Fatal("shouldPrefetch = false, want true for non-ambiguous structured impact with reads")
			}
			if policy.limit != tt.wantLimit {
				t.Fatalf("limit = %d, want %d", policy.limit, tt.wantLimit)
			}
			if policy.limited != tt.wantLimited {
				t.Fatalf("limited = %v, want %v", policy.limited, tt.wantLimited)
			}
			if policy.diagnostics != tt.diagnostics {
				t.Fatal("policy diagnostics did not carry metadata diagnostics")
			}
			for _, want := range tt.reasonContains {
				if !strings.Contains(policy.reason, want) {
					t.Fatalf("reason = %q, want fragment %q", policy.reason, want)
				}
			}
		})
	}
}

func TestPrefetchPolicyForArtifactKeepsStructuralGatesAndDiagnostics(t *testing.T) {
	diagnostics := &search.SymbolBundleDiagnostics{ResolvedBy: "fallback", Confidence: "low"}

	tests := []struct {
		name       string
		mutate     func(*search.SearchExecutionMetadata)
		wantReason string
	}{
		{
			name: "not structured",
			mutate: func(metadata *search.SearchExecutionMetadata) {
				metadata.StructuredImpact = false
			},
		},
		{
			name: "ambiguous",
			mutate: func(metadata *search.SearchExecutionMetadata) {
				metadata.Ambiguous = true
			},
			wantReason: "ambiguous structured impact",
		},
		{
			name: "nil bundle",
			mutate: func(metadata *search.SearchExecutionMetadata) {
				metadata.Bundle = nil
			},
		},
		{
			name: "nil impact",
			mutate: func(metadata *search.SearchExecutionMetadata) {
				metadata.Bundle.Impact = nil
			},
		},
		{
			name: "empty recommended reads",
			mutate: func(metadata *search.SearchExecutionMetadata) {
				metadata.Bundle.Impact.RecommendedReads = nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := prefetchPolicyMetadata(diagnostics)
			tt.mutate(&metadata)
			policy := prefetchPolicyForArtifact(metadata)
			if policy.shouldPrefetch {
				t.Fatal("shouldPrefetch = true, want false for gated prefetch")
			}
			if policy.diagnostics != diagnostics {
				t.Fatal("policy should carry metadata diagnostics even when prefetch is gated")
			}
			if policy.reason != tt.wantReason {
				t.Fatalf("reason = %q, want %q", policy.reason, tt.wantReason)
			}
		})
	}
}
