package search

import (
	"strings"
	"testing"
)

func TestSymbolBundleDiagnosticsSummary(t *testing.T) {
	tests := []struct {
		name string
		diag SymbolBundleDiagnostics
		want string
	}{
		{
			name: "zero diagnostics",
		},
		{
			name: "unknown only",
			diag: SymbolBundleDiagnostics{
				ResolvedBy:       symbolBundleResolvedByUnknown,
				Confidence:       symbolBundleConfidenceUnknown,
				LSPAttempted:     boolPtr(true),
				LSPAvailable:     boolPtr(false),
				LSPTimedOut:      boolPtr(false),
				RawRefCount:      intPtr(4),
				AcceptedRefCount: intPtr(3),
				DroppedRefCount:  intPtr(1),
			},
		},
		{
			name: "lsp high omits noisy metadata",
			diag: SymbolBundleDiagnostics{
				ResolvedBy:       symbolBundleResolvedByLSP,
				Confidence:       symbolBundleConfidenceHigh,
				LSPAttempted:     boolPtr(true),
				LSPAvailable:     boolPtr(true),
				LSPTimedOut:      boolPtr(false),
				RawRefCount:      intPtr(3),
				AcceptedRefCount: intPtr(2),
				DroppedRefCount:  intPtr(1),
			},
			want: "resolved_by=lsp, confidence=high",
		},
		{
			name: "mixed low fallback and budget flags",
			diag: SymbolBundleDiagnostics{
				ResolvedBy:     symbolBundleResolvedByMixed,
				Confidence:     symbolBundleConfidenceLow,
				FallbackReason: symbolBundleFallbackReasonLSPError,
				Truncated:      boolPtr(true),
				BudgetLimitHit: boolPtr(true),
			},
			want: "resolved_by=mixed, confidence=low, fallback_reason=lsp_error, truncated=true, budget_limit_hit=true",
		},
		{
			name: "legacy incomplete and truncated flags",
			diag: SymbolBundleDiagnostics{
				ResolvedBy:         symbolBundleResolvedByAST,
				Confidence:         symbolBundleConfidenceMedium,
				UpstreamIncomplete: true,
				UpstreamTruncated:  true,
				RawRefCount:        intPtr(10),
				AcceptedRefCount:   intPtr(5),
				DroppedRefCount:    intPtr(5),
				LSPAttempted:       boolPtr(false),
				LSPAvailable:       boolPtr(false),
				LSPTimedOut:        boolPtr(false),
			},
			want: "resolved_by=ast, confidence=medium, incomplete=true, truncated=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := symbolBundleDiagnosticsSummary(tt.diag); got != tt.want {
				t.Fatalf("summary = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatSymbolBundleDiagnosticsSummaryForStructuredImpact(t *testing.T) {
	tests := []struct {
		name       string
		diag       SymbolBundleDiagnostics
		want       []string
		wantAbsent []string
	}{
		{
			name: "LSP high",
			diag: SymbolBundleDiagnostics{
				ResolvedBy:       symbolBundleResolvedByLSP,
				Confidence:       symbolBundleConfidenceHigh,
				LSPAttempted:     boolPtr(true),
				LSPAvailable:     boolPtr(true),
				RawRefCount:      intPtr(2),
				AcceptedRefCount: intPtr(1),
			},
			want: []string{"Diagnostics: resolved_by=lsp, confidence=high"},
			wantAbsent: []string{
				"raw_ref_count",
				"accepted_ref_count",
				"dropped_ref_count",
				"lsp_attempted",
				"lsp_available",
			},
		},
		{
			name: "AST medium",
			diag: SymbolBundleDiagnostics{
				ResolvedBy: symbolBundleResolvedByAST,
				Confidence: symbolBundleConfidenceMedium,
			},
			want: []string{"Diagnostics: resolved_by=ast, confidence=medium"},
		},
		{
			name: "mixed low",
			diag: SymbolBundleDiagnostics{
				ResolvedBy:     symbolBundleResolvedByMixed,
				Confidence:     symbolBundleConfidenceLow,
				FallbackReason: symbolBundleFallbackReasonLSPError,
			},
			want: []string{"Diagnostics: resolved_by=mixed, confidence=low, fallback_reason=lsp_error"},
		},
		{
			name: "truncated and budget hit",
			diag: SymbolBundleDiagnostics{
				ResolvedBy:     symbolBundleResolvedByAST,
				Confidence:     symbolBundleConfidenceLow,
				Truncated:      boolPtr(true),
				BudgetLimitHit: boolPtr(true),
			},
			want: []string{"Diagnostics: resolved_by=ast, confidence=low, truncated=true, budget_limit_hit=true"},
		},
		{
			name: "unknown only",
			diag: SymbolBundleDiagnostics{
				ResolvedBy: symbolBundleResolvedByUnknown,
				Confidence: symbolBundleConfidenceUnknown,
			},
			wantAbsent: []string{"Diagnostics:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := formatSymbolBundle(testImpactDiagnosticsBundle(tt.diag), nil, nil)
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Fatalf("output missing %q:\n%s", want, output)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(output, absent) {
					t.Fatalf("output contains %q, want omitted:\n%s", absent, output)
				}
			}
		})
	}
}

func testImpactDiagnosticsBundle(diag SymbolBundleDiagnostics) *SymbolBundle {
	return &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Kind:        "function",
			DisplayName: "buildUser",
			File:        "src/build.ts",
			Line:        1,
		},
		Definition: SymbolBundleDefinition{
			Line:      1,
			Signature: "function buildUser() {}",
		},
		Impact: &SymbolBundleImpact{
			RecommendedReads: []SymbolBundleItem{{
				Kind:    "definition",
				File:    "src/build.ts",
				Line:    1,
				Snippet: "function buildUser() {}",
			}},
		},
		Diagnostics: diag,
	}
}

func assertDiagnosticsSummaryFragments(t *testing.T, output string, fragments ...string) {
	t.Helper()
	if !strings.Contains(output, "Diagnostics:") {
		t.Fatalf("output missing Diagnostics summary:\n%s", output)
	}
	for _, fragment := range fragments {
		if !strings.Contains(output, fragment) {
			t.Fatalf("Diagnostics summary missing %q:\n%s", fragment, output)
		}
	}
}
