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

func TestMultilineReader_BracketedPaste(t *testing.T) {
	// Bracketed paste format: ESC[200~...content...ESC[201~
	input := "\x1b[200~line 1\nline 2\nline 3\x1b[201~\n"
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

func TestMultilineReader_BracketedPaste_MultipleLines(t *testing.T) {
	// Simulate bracketed paste with multiple lines
	input := "\x1b[200~first line\nsecond line\nthird line\n\x1b[201~\n"
	reader := NewMultilineReader(strings.NewReader(input))

	result, err := reader.ReadInput("> ")
	if err != nil {
		t.Fatalf("ReadInput() error = %v", err)
	}

	lines := strings.Split(result, "\n")
	// Expected: "first line", "second line", "third line" (3 lines)
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d: %#v", len(lines), lines)
	}

	if lines[0] != "first line" {
		t.Errorf("Line 0 = %q, want %q", lines[0], "first line")
	}
	if lines[1] != "second line" {
		t.Errorf("Line 1 = %q, want %q", lines[1], "second line")
	}
	if lines[2] != "third line" {
		t.Errorf("Line 2 = %q, want %q", lines[2], "third line")
	}
}

func TestIsMultilineMarker(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"```", true},
		{"`` `", false},
		{"```\n", false},
		{" ```", false},
		{"test", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsMultilineMarker(tt.input)
			if got != tt.want {
				t.Errorf("IsMultilineMarker(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestTrimBracketedPasteMarkers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "with both markers",
			input: "\x1b[200~hello world\x1b[201~",
			want:  "hello world",
		},
		{
			name:  "with start marker only",
			input: "\x1b[200~hello world",
			want:  "hello world",
		},
		{
			name:  "with end marker only",
			input: "hello world\x1b[201~",
			want:  "hello world",
		},
		{
			name:  "no markers",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TrimBracketedPasteMarkers(tt.input)
			if got != tt.want {
				t.Errorf("TrimBracketedPasteMarkers() = %q, want %q", got, tt.want)
			}
		})
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

func TestMultilineReader_BracketedPaste_SingleLine(t *testing.T) {
	input := "\x1b[200~single line\x1b[201~\n"
	reader := NewMultilineReader(strings.NewReader(input))

	result, err := reader.ReadInput("> ")
	if err != nil {
		t.Fatalf("ReadInput() error = %v", err)
	}

	expected := "single line"
	if result != expected {
		t.Errorf("ReadInput() = %q, want %q", result, expected)
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
