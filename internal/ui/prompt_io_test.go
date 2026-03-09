package ui

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/stdio"
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

func TestDefaultPromptIOReusesSharedRuntimeReader(t *testing.T) {
	t.Cleanup(func() {
		stdio.SetDefaults(nil, nil, nil)
		DefaultRuntime().SetPromptReader(nil)
		DefaultRuntime().StopSpinner()
	})

	stdio.SetDefaults(strings.NewReader("yes\nno\n"), &bytes.Buffer{}, &bytes.Buffer{})
	DefaultRuntime().SetPromptReader(nil)

	firstPrompt := DefaultPromptIO()
	first, err := firstPrompt.ReadSimpleLine()
	if err != nil {
		t.Fatalf("read first line: %v", err)
	}
	if first != "yes" {
		t.Fatalf("first line = %q, want %q", first, "yes")
	}

	secondPrompt := DefaultPromptIO()
	second, err := secondPrompt.ReadSimpleLine()
	if err != nil {
		t.Fatalf("read second line: %v", err)
	}
	if second != "no" {
		t.Fatalf("second line = %q, want %q", second, "no")
	}
}

func TestDefaultPromptIOResetsSharedReaderWhenInputChanges(t *testing.T) {
	t.Cleanup(func() {
		stdio.SetDefaults(nil, nil, nil)
		DefaultRuntime().SetPromptReader(nil)
		DefaultRuntime().StopSpinner()
	})

	stdio.SetDefaults(strings.NewReader("old\nstale\n"), &bytes.Buffer{}, &bytes.Buffer{})
	DefaultRuntime().SetPromptReader(nil)

	firstPrompt := DefaultPromptIO()
	first, err := firstPrompt.ReadSimpleLine()
	if err != nil {
		t.Fatalf("read first line: %v", err)
	}
	if first != "old" {
		t.Fatalf("first line = %q, want %q", first, "old")
	}

	stdio.SetDefaults(strings.NewReader("fresh\n"), &bytes.Buffer{}, &bytes.Buffer{})

	secondPrompt := DefaultPromptIO()
	second, err := secondPrompt.ReadSimpleLine()
	if err != nil {
		t.Fatalf("read second line after input change: %v", err)
	}
	if second != "fresh" {
		t.Fatalf("second line = %q, want %q", second, "fresh")
	}
}
