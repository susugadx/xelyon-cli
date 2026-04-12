package agent

import "testing"

func TestJoinReadFileBatchSections(t *testing.T) {
	perFile := map[string]string{
		"a.go": "package a\nfunc A() {}",
		"b.go": "package b",
	}

	got, ok := joinReadFileBatchSections(perFile, []string{"a.go", "b.go"})
	if !ok {
		t.Fatal("joinReadFileBatchSections() = false, want true")
	}

	want := "📄 File: a.go\npackage a\nfunc A() {}\n📄 File: b.go\npackage b"
	if got != want {
		t.Fatalf("joinReadFileBatchSections() = %q, want %q", got, want)
	}
}

func TestJoinReadFileBatchSections_MissingSection(t *testing.T) {
	perFile := map[string]string{
		"a.go": "package a",
	}

	got, ok := joinReadFileBatchSections(perFile, []string{"a.go", "b.go"})
	if ok {
		t.Fatalf("joinReadFileBatchSections() = (%q, true), want false", got)
	}
}
