package gathercontext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

	prefetch := prefetchRecommendedEvidence(newGatherContextExecCtx(root), artifact)
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

	execCtx := newGatherContextExecCtx(root)
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
