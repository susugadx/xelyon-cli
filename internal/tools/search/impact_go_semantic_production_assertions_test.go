package search

import (
	"reflect"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func assertDiagnosticsEqualToGoSourceOfTruth(t *testing.T, result navigation.InspectResult, bundle *SymbolBundle) {
	t.Helper()
	expected := &SymbolBundle{
		Sections:    append([]SymbolBundleSection(nil), bundle.Sections...),
		Diagnostics: goSymbolBundleDiagnostics(result),
	}
	finalizeSymbolBundleDiagnostics(expected)
	if !reflect.DeepEqual(bundle.Diagnostics, expected.Diagnostics) {
		t.Fatalf("diagnostics = %+v, want goSymbolBundleDiagnostics/finalize result %+v", bundle.Diagnostics, expected.Diagnostics)
	}
}

func assertGoStructuredProductionSectionOrder(t *testing.T, bundle *SymbolBundle, kinds ...string) {
	t.Helper()
	got := make([]string, 0, len(bundle.Sections))
	for _, section := range bundle.Sections {
		got = append(got, section.Kind)
	}
	if !reflect.DeepEqual(got, kinds) {
		t.Fatalf("section order = %v, want %v", got, kinds)
	}
}

func assertGoStructuredProductionRecommendedReadKinds(t *testing.T, reads []SymbolBundleItem, kinds ...string) {
	t.Helper()
	if len(reads) != len(kinds) {
		t.Fatalf("RecommendedReads length = %d, want %d; reads = %+v", len(reads), len(kinds), reads)
	}
	for i, want := range kinds {
		if reads[i].Kind != want {
			t.Fatalf("RecommendedReads[%d].Kind = %q, want %q; reads = %+v", i, reads[i].Kind, want, reads)
		}
	}
}
