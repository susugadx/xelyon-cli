package search

import (
	"strings"
	"testing"
)

func assertSemanticReferenceKindCount(t *testing.T, refs []SemanticReference, kind string, want int) {
	t.Helper()
	got := 0
	for _, ref := range refs {
		if ref.Kind == kind {
			got++
		}
	}
	if got != want {
		t.Fatalf("reference kind %q count = %d, want %d; refs = %+v", kind, got, want, refs)
	}
}

func assertSymbolBundleSectionSummary(t *testing.T, bundle *SymbolBundle, kind string, wantTotal int, wantMore bool) {
	t.Helper()
	section := symbolBundleSectionByKind(bundle, kind)
	if section == nil {
		t.Fatalf("section %q = nil, want total %d with More %v", kind, wantTotal, wantMore)
	}
	if section.Total != wantTotal || section.More != wantMore {
		t.Fatalf("section %q total/more = %d/%v, want %d/%v; section = %+v", kind, section.Total, section.More, wantTotal, wantMore, section)
	}
}

func assertSymbolBundleSectionItemCount(t *testing.T, bundle *SymbolBundle, kind string, want int) {
	t.Helper()
	section := symbolBundleSectionByKind(bundle, kind)
	if section == nil {
		t.Fatalf("section %q = nil, want %d items", kind, want)
	}
	if got := len(section.Items); got != want {
		t.Fatalf("section %q item count = %d, want %d; section = %+v", kind, got, want, section)
	}
}

func assertOutputContains(t *testing.T, output, fragment string) {
	t.Helper()
	if !strings.Contains(output, fragment) {
		t.Fatalf("expected output to contain %q, got:\n%s", fragment, output)
	}
}
