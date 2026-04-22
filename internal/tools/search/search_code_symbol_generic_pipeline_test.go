package search

import (
	"bytes"
	"testing"
)

func TestParseGenericSymbolMatchLine(t *testing.T) {
	match, ok := parseGenericSymbolMatchLine("pkg/service.go:42:  func Run() error {  ", SearchOptions{})
	if !ok {
		t.Fatal("expected parseGenericSymbolMatchLine to parse valid rg output")
	}
	if match.File != "pkg/service.go" {
		t.Fatalf("file = %q, want %q", match.File, "pkg/service.go")
	}
	if match.Line != 42 {
		t.Fatalf("line = %d, want %d", match.Line, 42)
	}
	if match.Content != "func Run() error {" {
		t.Fatalf("content = %q, want %q", match.Content, "func Run() error {")
	}
}

func TestParseGenericSymbolMatches_AppliesFilterAndLimit(t *testing.T) {
	var stdout bytes.Buffer
	stdout.WriteString("pkg/service.go:10:func Run(){}\n")
	stdout.WriteString("pkg/service.py:11:def Run():\n")
	stdout.WriteString("pkg/runner.go:12:Run()\n")

	matches := parseGenericSymbolMatches(stdout, SearchOptions{FileType: "go"}, 1)
	if len(matches) != 1 {
		t.Fatalf("matches len = %d, want %d", len(matches), 1)
	}
	if matches[0].File != "pkg/service.go" || matches[0].Line != 10 {
		t.Fatalf("unexpected first match: %+v", matches[0])
	}
}
