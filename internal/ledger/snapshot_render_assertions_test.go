package ledger

import (
	"strings"
	"testing"
)

func assertSnapshotRenderEquals(t *testing.T, got string, wantLines ...string) {
	t.Helper()
	want := snapshotText(wantLines...)
	if got != want {
		t.Fatalf("snapshot =\n%s\nwant =\n%s", got, want)
	}
}

func snapshotText(lines ...string) string {
	return strings.Join(lines, "\n")
}

func assertSnapshotRenderContains(t *testing.T, output string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(output, fragment) {
			t.Fatalf("snapshot missing %q:\n%s", fragment, output)
		}
	}
}

func assertSnapshotRenderOmits(t *testing.T, output string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if strings.Contains(output, fragment) {
			t.Fatalf("snapshot should not contain %q:\n%s", fragment, output)
		}
	}
}
