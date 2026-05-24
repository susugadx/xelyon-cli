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

func TestPrefetchRecommendedEvidence_BoundsReadsAndKeepsOrder(t *testing.T) {
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

func TestPrefetchPolicyForArtifactCarriesDiagnosticsAndKeepsAmbiguousGate(t *testing.T) {
	diagnostics := &search.SymbolBundleDiagnostics{ResolvedBy: "fallback", Confidence: "low"}
	metadata := search.SearchExecutionMetadata{
		StructuredImpact: true,
		Diagnostics:      diagnostics,
		Bundle: &search.SymbolBundle{
			Impact: &search.SymbolBundleImpact{
				RecommendedReads: []search.SymbolBundleItem{{Kind: "definition", File: "valid.go", Line: 1}},
			},
		},
	}

	policy := prefetchPolicyForArtifact(metadata)
	if !policy.shouldPrefetch {
		t.Fatal("shouldPrefetch = false, want true for non-ambiguous structured impact with reads")
	}
	if policy.diagnostics != diagnostics {
		t.Fatal("policy diagnostics did not carry metadata diagnostics")
	}

	metadata.Ambiguous = true
	policy = prefetchPolicyForArtifact(metadata)
	if policy.shouldPrefetch {
		t.Fatal("shouldPrefetch = true, want false for ambiguous structured impact")
	}
	if policy.diagnostics != diagnostics {
		t.Fatal("ambiguous policy should still receive diagnostics for future gates")
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
