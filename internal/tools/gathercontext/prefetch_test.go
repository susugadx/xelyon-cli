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

	output := prefetchRecommendedEvidence(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, artifact)

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
	if ambiguous != "" {
		t.Fatalf("expected ambiguous structured impact to skip prefetch, got:\n%s", ambiguous)
	}

	output := prefetchRecommendedEvidence(execCtx, search.SearchExecutionArtifact{
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
	if !strings.Contains(output, "valid.go") {
		t.Fatalf("expected successful prefetched reads to survive missing files, got:\n%s", output)
	}
	if strings.Contains(output, "missing.go") || strings.Contains(output, "Error:") {
		t.Fatalf("expected failed reads to be skipped silently, got:\n%s", output)
	}
}
