package gathercontext

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

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
