package readtool

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

func TestRenderReadFilesResults_WithoutLocator(t *testing.T) {
	results := []readFileBatchResult{
		{entry: "a.go", filePath: "a.go", result: "1: package main\n"},
		{entry: "b.go", filePath: "b.go", result: "1: package util\n"},
	}

	rendered := renderReadFilesResults(results, nil)
	if !strings.Contains(rendered, "📄 File: a.go") {
		t.Fatalf("expected first header, got: %s", rendered)
	}
	if !strings.Contains(rendered, "📄 File: b.go") {
		t.Fatalf("expected second header, got: %s", rendered)
	}
	if !strings.Contains(rendered, "1: package util") {
		t.Fatalf("expected content, got: %s", rendered)
	}
}

func TestRenderReadFilesResults_WithLocator(t *testing.T) {
	reg := locator.NewRegistry()
	results := []readFileBatchResult{
		{
			entry:        "a.go:10-20",
			filePath:     "a.go",
			resolvedPath: "/tmp/real/a.go",
			startLine:    10,
			endLine:      20,
			result:       "10: line10\n",
		},
	}

	rendered := renderReadFilesResults(results, reg)
	if !strings.Contains(rendered, "📄 File: a.go:10-20 [L1]") {
		t.Fatalf("expected locator in header, got: %s", rendered)
	}

	loc, ok := reg.Resolve("[L1]")
	if !ok {
		t.Fatal("expected locator to be registered")
	}
	if loc.FilePath != "a.go" || loc.ResolvedPath != "/tmp/real/a.go" || loc.Line != 10 || loc.EndLine != 20 {
		t.Fatalf("unexpected locator: %+v", loc)
	}
}
