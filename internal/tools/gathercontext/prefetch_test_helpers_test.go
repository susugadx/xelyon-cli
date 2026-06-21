package gathercontext

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

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
