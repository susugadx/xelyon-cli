package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/toolruntime"
)

func TestJoinReadFileBatchSections(t *testing.T) {
	perFile := map[string]string{
		"a.go": "package a\nfunc A() {}",
		"b.go": "package b",
	}

	got, ok := toolruntime.JoinReadFileBatchSections(perFile, []string{"a.go", "b.go"})
	if !ok {
		t.Fatal("toolruntime.JoinReadFileBatchSections() = false, want true")
	}

	want := "📄 File: a.go\npackage a\nfunc A() {}\n📄 File: b.go\npackage b"
	if got != want {
		t.Fatalf("toolruntime.JoinReadFileBatchSections() = %q, want %q", got, want)
	}
}

func TestJoinReadFileBatchSections_MissingSection(t *testing.T) {
	perFile := map[string]string{
		"a.go": "package a",
	}

	got, ok := toolruntime.JoinReadFileBatchSections(perFile, []string{"a.go", "b.go"})
	if ok {
		t.Fatalf("toolruntime.JoinReadFileBatchSections() = (%q, true), want false", got)
	}
}
