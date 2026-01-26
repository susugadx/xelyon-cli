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

func TestStripAllBracketedPasteMarkers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "ESC markers",
			input: "\x1b[200~hello\x1b[201~",
			want:  "hello",
		},
		{
			name:  "literal markers",
			input: "^[[200~hello^[[201~",
			want:  "hello",
		},
		{
			name:  "mixed markers",
			input: "\x1b[200~hello^[[201~",
			want:  "hello",
		},
		{
			name:  "multiple markers",
			input: "^[[200~a^[[200~b^[[201~c^[[201~",
			want:  "abc",
		},
		{
			name:  "no markers",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripAllBracketedPasteMarkers(tt.input)
			if got != tt.want {
				t.Errorf("stripAllBracketedPasteMarkers() = %q, want %q", got, tt.want)
			}
		})
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

func TestMultilineReader_DisableBracketedPaste(t *testing.T) {
	reader := NewMultilineReader(strings.NewReader("test\n"))

	// DisableBracketedPaste should be safe to call even when not enabled
	reader.DisableBracketedPaste()
	if reader.IsBracketedPasteEnabled() {
		t.Error("IsBracketedPasteEnabled() should be false after disable")
	}
}
