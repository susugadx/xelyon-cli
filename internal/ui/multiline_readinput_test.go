package ui

import (
	"strings"
	"testing"
)

func TestMultilineReader_SingleLine(t *testing.T) {
	input := "Hello World\n"
	reader := NewMultilineReader(strings.NewReader(input))

	result, err := reader.ReadInput("> ")
	if err != nil {
		t.Fatalf("ReadInput() error = %v", err)
	}

	expected := "Hello World"
	if result != expected {
		t.Errorf("ReadInput() = %q, want %q", result, expected)
	}
}

func TestMultilineReader_EmptyLine(t *testing.T) {
	input := "\n"
	reader := NewMultilineReader(strings.NewReader(input))

	result, err := reader.ReadInput("> ")
	if err != nil {
		t.Fatalf("ReadInput() error = %v", err)
	}

	if result != "" {
		t.Errorf("ReadInput() = %q, want empty string", result)
	}
}

func TestMultilineReader_MarkerMode(t *testing.T) {
	input := "```\nline 1\nline 2\nline 3\n```\n"
	reader := NewMultilineReader(strings.NewReader(input))

	result, err := reader.ReadInput("> ")
	if err != nil {
		t.Fatalf("ReadInput() error = %v", err)
	}

	expected := "line 1\nline 2\nline 3"
	if result != expected {
		t.Errorf("ReadInput() = %q, want %q", result, expected)
	}
}

func TestMultilineReader_MarkerMode_EmptyLines(t *testing.T) {
	input := "```\n\nline 1\n\nline 2\n\n```\n"
	reader := NewMultilineReader(strings.NewReader(input))

	result, err := reader.ReadInput("> ")
	if err != nil {
		t.Fatalf("ReadInput() error = %v", err)
	}

	expected := "\nline 1\n\nline 2\n"
	if result != expected {
		t.Errorf("ReadInput() = %q, want %q", result, expected)
	}
}

func TestMultilineReader_MarkerMode_OnlyMarkers(t *testing.T) {
	input := "```\n```\n"
	reader := NewMultilineReader(strings.NewReader(input))

	result, err := reader.ReadInput("> ")
	if err != nil {
		t.Fatalf("ReadInput() error = %v", err)
	}

	if result != "" {
		t.Errorf("ReadInput() = %q, want empty string", result)
	}
}

func TestMultilineReader_MarkerMode_WithCode(t *testing.T) {
	input := "```\npackage main\n\nfunc main() {\n    println(\"hello\")\n}\n```\n"
	reader := NewMultilineReader(strings.NewReader(input))

	result, err := reader.ReadInput("> ")
	if err != nil {
		t.Fatalf("ReadInput() error = %v", err)
	}

	expected := "package main\n\nfunc main() {\n    println(\"hello\")\n}"
	if result != expected {
		t.Errorf("ReadInput() = %q, want %q", result, expected)
	}
}

func TestMultilineReader_BracketedPaste_ESC(t *testing.T) {
	// Bracketed paste format: ESC[200~...content...ESC[201~ (single line, markers stripped)
	input := "\x1b[200~hello world\x1b[201~\n"
	reader := NewMultilineReader(strings.NewReader(input))

	result, err := reader.ReadInput("> ")
	if err != nil {
		t.Fatalf("ReadInput() error = %v", err)
	}

	expected := "hello world"
	if result != expected {
		t.Errorf("ReadInput() = %q, want %q", result, expected)
	}
}

func TestMultilineReader_BracketedPaste_Literal(t *testing.T) {
	// Literal form: ^[[200~...content...^[[201~ (stripped)
	input := "^[[200~hello world^[[201~\n"
	reader := NewMultilineReader(strings.NewReader(input))

	result, err := reader.ReadInput("> ")
	if err != nil {
		t.Fatalf("ReadInput() error = %v", err)
	}

	expected := "hello world"
	if result != expected {
		t.Errorf("ReadInput() = %q, want %q", result, expected)
	}
}

func TestMultilineReader_BracketedPaste_PartialMarkers(t *testing.T) {
	// Only start marker present (literal form)
	input := "^[[200~hello world\n"
	reader := NewMultilineReader(strings.NewReader(input))

	result, err := reader.ReadInput("> ")
	if err != nil {
		t.Fatalf("ReadInput() error = %v", err)
	}

	expected := "hello world"
	if result != expected {
		t.Errorf("ReadInput() = %q, want %q", result, expected)
	}
}
