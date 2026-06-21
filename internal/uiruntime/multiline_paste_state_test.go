package uiruntime

import (
	"bytes"
	"strings"
	"testing"
)

func TestMultilineReader_IsBracketedPasteEnabled(t *testing.T) {
	reader := NewMultilineReader(strings.NewReader("test\n"))

	// Initially disabled (not a terminal)
	if reader.IsBracketedPasteEnabled() {
		t.Error("IsBracketedPasteEnabled() should be false initially for non-terminal")
	}

	// EnableBracketedPaste should not enable for non-terminal
	reader.EnableBracketedPaste()
	if reader.IsBracketedPasteEnabled() {
		t.Error("IsBracketedPasteEnabled() should remain false for non-terminal")
	}
}

func TestMultilineReader_EnableBracketedPaste_UsesRuntimeErrorOutput(t *testing.T) {
	t.Setenv("XELYON_DEBUG_PASTE", "1")

	var out bytes.Buffer
	var errOut bytes.Buffer
	reader := NewMultilineReaderWithRuntime(NewRuntime(strings.NewReader("test\n"), &out, &errOut))

	reader.EnableBracketedPaste()

	if !strings.Contains(errOut.String(), "EnableBracketedPaste") {
		t.Fatalf("expected runtime error output to contain debug message, got %q", errOut.String())
	}
	if strings.Contains(out.String(), "EnableBracketedPaste") {
		t.Fatalf("debug message should not leak to stdout buffer, got %q", out.String())
	}
}

func TestMultilineReader_DisableBracketedPaste(t *testing.T) {
	reader := NewMultilineReader(strings.NewReader("test\n"))

	// DisableBracketedPaste should be safe to call even when not enabled
	reader.DisableBracketedPaste()
	if reader.IsBracketedPasteEnabled() {
		t.Error("IsBracketedPasteEnabled() should be false after disable")
	}
}
