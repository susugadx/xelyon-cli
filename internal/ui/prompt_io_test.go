package ui

import (
	"io"
	"strings"
	"testing"
)

func TestPromptIOReadSimpleLineReusesBufferedReader(t *testing.T) {
	promptIO := NewPromptIO(strings.NewReader("first\nsecond\n"), io.Discard, io.Discard, nil)

	first, err := promptIO.ReadSimpleLine()
	if err != nil {
		t.Fatalf("read first line: %v", err)
	}
	if first != "first" {
		t.Fatalf("first line = %q, want %q", first, "first")
	}

	second, err := promptIO.ReadSimpleLine()
	if err != nil {
		t.Fatalf("read second line: %v", err)
	}
	if second != "second" {
		t.Fatalf("second line = %q, want %q", second, "second")
	}
}

func TestNormalizePromptIOPreservesBufferedReader(t *testing.T) {
	promptIO := NewPromptIO(strings.NewReader("alpha\nbeta\n"), io.Discard, io.Discard, nil)

	first, err := promptIO.ReadSimpleLine()
	if err != nil {
		t.Fatalf("read first line: %v", err)
	}
	if first != "alpha" {
		t.Fatalf("first line = %q, want %q", first, "alpha")
	}

	normalized := NormalizePromptIO(promptIO)
	second, err := normalized.ReadSimpleLine()
	if err != nil {
		t.Fatalf("read second line after normalize: %v", err)
	}
	if second != "beta" {
		t.Fatalf("second line = %q, want %q", second, "beta")
	}
}
