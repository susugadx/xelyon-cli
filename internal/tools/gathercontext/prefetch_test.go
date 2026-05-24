package gathercontext

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

func TestPrefetchRecommendedEvidence_NilDiagnosticsKeepCompatibilityLimitAndOrder(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	files := []string{"alpha.go", "beta.go", "gamma.go", "delta.go"}
	for _, name := range files {
		content := "package sample\n\nconst value = \"" + name + "\"\n"
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	artifact := search.SearchExecutionArtifact{
		Metadata: search.SearchExecutionMetadata{
			StructuredImpact: true,
			Bundle: &search.SymbolBundle{
				Impact: &search.SymbolBundleImpact{
					RecommendedReads: []search.SymbolBundleItem{
						{Kind: "definition", File: "alpha.go", ResolvedPath: filepath.Join(root, "alpha.go"), Line: 3, EndLine: 3, Name: "Alpha"},
						{Kind: "callers", File: "beta.go", ResolvedPath: filepath.Join(root, "beta.go"), Line: 3, EndLine: 3, Name: "Beta"},
						{Kind: "tests", File: "gamma.go", ResolvedPath: filepath.Join(root, "gamma.go"), Line: 3, EndLine: 3, Name: "Gamma"},
						{Kind: "references", File: "delta.go", ResolvedPath: filepath.Join(root, "delta.go"), Line: 3, EndLine: 3, Name: "Delta"},
					},
				},
			},
		},
	}

	prefetch := prefetchRecommendedEvidence(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, artifact)
	output := prefetch.output

	if count := strings.Count(output, "📄 File: "); count != 3 {
		t.Fatalf("expected bounded prefetch of 3 files, got %d:\n%s", count, output)
	}
	for _, want := range []string{"alpha.go", "beta.go", "gamma.go"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected prefetched output to include %s, got:\n%s", want, output)
		}
	}
	if strings.Contains(output, "delta.go") {
		t.Fatalf("expected prefetch to stop after top 3 reads, got:\n%s", output)
	}

	alpha := strings.Index(output, "alpha.go")
	beta := strings.Index(output, "beta.go")
	gamma := strings.Index(output, "gamma.go")
	if alpha < 0 || beta <= alpha || gamma <= beta {
		t.Fatalf("expected prefetched reads to keep ranked order, got:\n%s", output)
	}
	for _, want := range []string{"alpha.go", "beta.go", "gamma.go"} {
		assertObservationHasEvidence(t, prefetch.observation, filepath.Join(root, want), 3)
	}
	assertObservationMissingResolvedPath(t, prefetch.observation, filepath.Join(root, "delta.go"))
}

func TestPrefetchRecommendedEvidence_LimitsReadsFromDiagnostics(t *testing.T) {
	tests := []struct {
		name         string
		diagnostics  *search.SymbolBundleDiagnostics
		wantCount    int
		wantFiles    []string
		omittedFiles []string
		wantNote     []string
		noNote       bool
	}{
		{
			name: "high LSP diagnostics reads three",
			diagnostics: &search.SymbolBundleDiagnostics{
				ResolvedBy: search.SymbolBundleResolvedByLSP,
				Confidence: search.SymbolBundleConfidenceHigh,
			},
			wantCount: 3,
			wantFiles: []string{"alpha.go", "beta.go", "gamma.go"},
			noNote:    true,
		},
		{
			name: "medium AST diagnostics reads two with limited note",
			diagnostics: &search.SymbolBundleDiagnostics{
				ResolvedBy: search.SymbolBundleResolvedByAST,
				Confidence: search.SymbolBundleConfidenceMedium,
			},
			wantCount:    2,
			wantFiles:    []string{"alpha.go", "beta.go"},
			omittedFiles: []string{"gamma.go"},
			wantNote:     []string{"Prefetch limited:", "confidence=medium"},
		},
		{
			name: "low fallback diagnostics reads one with fallback note",
			diagnostics: &search.SymbolBundleDiagnostics{
				ResolvedBy:     search.SymbolBundleResolvedByFallback,
				Confidence:     search.SymbolBundleConfidenceLow,
				FallbackReason: search.SymbolBundleFallbackReasonStructuredUnavailable,
			},
			wantCount:    1,
			wantFiles:    []string{"alpha.go"},
			omittedFiles: []string{"beta.go", "gamma.go"},
			wantNote:     []string{"Prefetch limited:", "resolved_by=fallback", "confidence=low", "fallback_reason=structured_unavailable"},
		},
		{
			name: "mixed truncated diagnostics reads one with fallback reason note",
			diagnostics: &search.SymbolBundleDiagnostics{
				ResolvedBy:     search.SymbolBundleResolvedByMixed,
				Confidence:     search.SymbolBundleConfidenceLow,
				Truncated:      prefetchBoolPtr(true),
				FallbackReason: search.SymbolBundleFallbackReasonLSPError,
			},
			wantCount:    1,
			wantFiles:    []string{"alpha.go"},
			omittedFiles: []string{"beta.go", "gamma.go"},
			wantNote:     []string{"Prefetch limited:", "truncated=true", "resolved_by=mixed", "fallback_reason=lsp_error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			withGatherContextWorkingDir(t, root)

			files := []string{"alpha.go", "beta.go", "gamma.go"}
			for _, name := range files {
				content := "package sample\n\nconst value = \"" + name + "\"\n"
				if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			prefetch := prefetchRecommendedEvidence(newGatherContextExecCtx(root), search.SearchExecutionArtifact{
				Metadata: search.SearchExecutionMetadata{
					StructuredImpact: true,
					Diagnostics:      tt.diagnostics,
					Bundle: &search.SymbolBundle{
						Impact: &search.SymbolBundleImpact{
							RecommendedReads: []search.SymbolBundleItem{
								{Kind: "definition", File: "alpha.go", ResolvedPath: filepath.Join(root, "alpha.go"), Line: 3, EndLine: 3, Name: "Alpha"},
								{Kind: "callers", File: "beta.go", ResolvedPath: filepath.Join(root, "beta.go"), Line: 3, EndLine: 3, Name: "Beta"},
								{Kind: "tests", File: "gamma.go", ResolvedPath: filepath.Join(root, "gamma.go"), Line: 3, EndLine: 3, Name: "Gamma"},
							},
						},
					},
				},
			})
			output := prefetch.output

			if count := strings.Count(output, "📄 File: "); count != tt.wantCount {
				t.Fatalf("prefetched read count = %d, want %d:\n%s", count, tt.wantCount, output)
			}
			for _, want := range tt.wantFiles {
				if !strings.Contains(output, want) {
					t.Fatalf("expected %s in prefetched output, got:\n%s", want, output)
				}
			}
			for _, omitted := range tt.omittedFiles {
				if strings.Contains(output, omitted) {
					t.Fatalf("did not expect %s in limited prefetched output, got:\n%s", omitted, output)
				}
			}
			for _, want := range tt.wantNote {
				if !strings.Contains(output, want) {
					t.Fatalf("expected note fragment %q in prefetched output, got:\n%s", want, output)
				}
			}
			if tt.noNote && strings.Contains(output, "Prefetch limited:") {
				t.Fatalf("did not expect limited note for high confidence prefetch, got:\n%s", output)
			}
		})
	}
}

func TestPrefetchRecommendedEvidence_IgnoresAmbiguousOrFailedReads(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "valid.go"), []byte("package sample\n\nconst value = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	ambiguous := prefetchRecommendedEvidence(execCtx, search.SearchExecutionArtifact{
		Metadata: search.SearchExecutionMetadata{
			StructuredImpact: true,
			Ambiguous:        true,
			Bundle: &search.SymbolBundle{
				Impact: &search.SymbolBundleImpact{
					RecommendedReads: []search.SymbolBundleItem{
						{Kind: "definition", File: "valid.go", ResolvedPath: filepath.Join(root, "valid.go"), Line: 3, EndLine: 3, Name: "Valid"},
					},
				},
			},
		},
	})
	if ambiguous.output != "" || ambiguous.observation != nil {
		t.Fatalf("expected ambiguous structured impact to skip prefetch, got output:\n%s observation:%#v", ambiguous.output, ambiguous.observation)
	}
	if ambiguous.discoveryNote != "Prefetch skipped: ambiguous structured impact" {
		t.Fatalf("ambiguous discovery note = %q, want skip note", ambiguous.discoveryNote)
	}

	prefetch := prefetchRecommendedEvidence(execCtx, search.SearchExecutionArtifact{
		Metadata: search.SearchExecutionMetadata{
			StructuredImpact: true,
			Bundle: &search.SymbolBundle{
				Impact: &search.SymbolBundleImpact{
					RecommendedReads: []search.SymbolBundleItem{
						{Kind: "definition", File: "missing.go", ResolvedPath: filepath.Join(root, "missing.go"), Line: 3, EndLine: 3, Name: "Missing"},
						{Kind: "callers", File: "valid.go", ResolvedPath: filepath.Join(root, "valid.go"), Line: 3, EndLine: 3, Name: "Valid"},
					},
				},
			},
		},
	})
	output := prefetch.output
	if !strings.Contains(output, "valid.go") {
		t.Fatalf("expected successful prefetched reads to survive missing files, got:\n%s", output)
	}
	if strings.Contains(output, "missing.go") || strings.Contains(output, "Error:") {
		t.Fatalf("expected failed reads to be skipped silently, got:\n%s", output)
	}
	assertObservationHasEvidence(t, prefetch.observation, filepath.Join(root, "valid.go"), 3)
	assertObservationMissingResolvedPath(t, prefetch.observation, filepath.Join(root, "missing.go"))
}

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

func TestBuildSearchExecutionResult_MergesPrefetchedObservation(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "definition.go"): "package sample\n\ntype Builder struct{}\n",
		filepath.Join(root, "caller.go"):     "package sample\n\nfunc Use() { _ = Builder{} }\n",
	})

	result := buildSearchExecutionResult(
		newGatherContextExecCtx(root),
		searchPlan{query: "Builder", preferImpact: true},
		search.SearchExecutionArtifact{
			Rendered: "search discovery",
			Metadata: search.SearchExecutionMetadata{
				StructuredImpact: true,
				Observation: &tools.RuntimeObservation{
					TouchedFiles: []tools.ObservationPath{{
						Path:         "definition.go",
						ResolvedPath: filepath.Join(root, "definition.go"),
					}},
					Evidence: []tools.ObservationEvidence{{
						Path:         "definition.go",
						ResolvedPath: filepath.Join(root, "definition.go"),
						StartLine:    3,
						EndLine:      3,
						Excerpt:      "type Builder struct{}",
					}},
				},
				Bundle: &search.SymbolBundle{
					Impact: &search.SymbolBundleImpact{
						RecommendedReads: []search.SymbolBundleItem{{
							Kind:         "callers",
							File:         "caller.go",
							ResolvedPath: filepath.Join(root, "caller.go"),
							Line:         3,
							EndLine:      3,
							Name:         "Use",
						}},
					},
				},
			},
		},
	)

	if result.routeHint != "Structured impact + prefetched evidence" {
		t.Fatalf("routeHint = %q, want prefetched route", result.routeHint)
	}
	if result.search == nil || !strings.Contains(result.search.prefetchedEvidence, "caller.go") {
		t.Fatalf("prefetchedEvidence missing caller.go: %#v", result.search)
	}
	assertObservationHasEvidence(t, result.observation, filepath.Join(root, "definition.go"), 3)
	assertObservationHasEvidence(t, result.observation, filepath.Join(root, "caller.go"), 3)
}

func assertObservationHasEvidence(t *testing.T, observation *tools.RuntimeObservation, resolvedPath string, line int) {
	t.Helper()
	if observation == nil {
		t.Fatalf("Observation = nil, want evidence for %s:%d", resolvedPath, line)
	}
	touched := false
	for _, file := range observation.TouchedFiles {
		if file.ResolvedPath == resolvedPath {
			touched = true
			break
		}
	}
	if !touched {
		t.Fatalf("TouchedFiles = %#v, want %s", observation.TouchedFiles, resolvedPath)
	}
	for _, evidence := range observation.Evidence {
		if evidence.ResolvedPath == resolvedPath && evidence.StartLine == line && evidence.EndLine >= line {
			return
		}
	}
	t.Fatalf("Evidence = %#v, want %s:%d", observation.Evidence, resolvedPath, line)
}

func assertObservationMissingResolvedPath(t *testing.T, observation *tools.RuntimeObservation, resolvedPath string) {
	t.Helper()
	if observation == nil {
		return
	}
	for _, file := range observation.TouchedFiles {
		if file.ResolvedPath == resolvedPath {
			t.Fatalf("TouchedFiles = %#v, did not want %s", observation.TouchedFiles, resolvedPath)
		}
	}
	for _, evidence := range observation.Evidence {
		if evidence.ResolvedPath == resolvedPath {
			t.Fatalf("Evidence = %#v, did not want %s", observation.Evidence, resolvedPath)
		}
	}
}

func prefetchPolicyMetadata(diagnostics *search.SymbolBundleDiagnostics) search.SearchExecutionMetadata {
	return search.SearchExecutionMetadata{
		StructuredImpact: true,
		Diagnostics:      diagnostics,
		Bundle: &search.SymbolBundle{
			Impact: &search.SymbolBundleImpact{
				RecommendedReads: []search.SymbolBundleItem{{Kind: "definition", File: "valid.go", Line: 1}},
			},
		},
	}
}

func prefetchBoolPtr(value bool) *bool {
	return &value
}
