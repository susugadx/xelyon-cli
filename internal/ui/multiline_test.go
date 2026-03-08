package ui

import (
	"bytes"
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

// newTestReaderWithChannel creates a MultilineReader with channels for testing
func newTestReaderWithChannel() (*MultilineReader, chan byte, chan error) {
	r := NewMultilineReader(strings.NewReader(""))
	r.byteChan = make(chan byte, 4096)
	r.errChan = make(chan error, 1)
	r.rawModeInit = true
	return r, r.byteChan, r.errChan
}

// feedBytes sends bytes to the channel followed by Enter (\r)
func feedBytes(ch chan byte, data []byte) {
	for _, b := range data {
		ch <- b
	}
	ch <- '\r' // Enter to finish
}

func TestReadLineFromChannel_BackspaceASCII(t *testing.T) {
	r, ch, _ := newTestReaderWithChannel()

	// Type "abc" then backspace then Enter → "ab"
	go feedBytes(ch, []byte{'a', 'b', 'c', 0x7f})

	result, err := r.readLineFromChannel()
	if err != nil {
		t.Fatalf("readLineFromChannel() error = %v", err)
	}
	if result != "ab" {
		t.Errorf("readLineFromChannel() = %q, want %q", result, "ab")
	}
}

func TestReadLineFromChannel_BackspaceMultiple(t *testing.T) {
	r, ch, _ := newTestReaderWithChannel()

	// Type "hello" then 3 backspaces → "he"
	go feedBytes(ch, []byte{'h', 'e', 'l', 'l', 'o', 0x7f, 0x7f, 0x7f})

	result, err := r.readLineFromChannel()
	if err != nil {
		t.Fatalf("readLineFromChannel() error = %v", err)
	}
	if result != "he" {
		t.Errorf("readLineFromChannel() = %q, want %q", result, "he")
	}
}

func TestReadLineFromChannel_BackspaceEmpty(t *testing.T) {
	r, ch, _ := newTestReaderWithChannel()

	// Backspace on empty buffer should not crash
	go feedBytes(ch, []byte{0x7f, 0x7f, 'a'})

	result, err := r.readLineFromChannel()
	if err != nil {
		t.Fatalf("readLineFromChannel() error = %v", err)
	}
	if result != "a" {
		t.Errorf("readLineFromChannel() = %q, want %q", result, "a")
	}
}

func TestReadLineFromChannel_BackspaceAll(t *testing.T) {
	r, ch, _ := newTestReaderWithChannel()

	// Type "abc" then delete all → ""
	go feedBytes(ch, []byte{'a', 'b', 'c', 0x7f, 0x7f, 0x7f})

	result, err := r.readLineFromChannel()
	if err != nil {
		t.Fatalf("readLineFromChannel() error = %v", err)
	}
	if result != "" {
		t.Errorf("readLineFromChannel() = %q, want %q", result, "")
	}
}

func TestReadLineFromChannel_BackspaceUTF8(t *testing.T) {
	r, ch, _ := newTestReaderWithChannel()

	// Type "あa" (3-byte UTF-8 + ASCII) then backspace → "あ"
	// あ = E3 81 82
	go feedBytes(ch, []byte{0xE3, 0x81, 0x82, 'a', 0x7f})

	result, err := r.readLineFromChannel()
	if err != nil {
		t.Fatalf("readLineFromChannel() error = %v", err)
	}
	if result != "あ" {
		t.Errorf("readLineFromChannel() = %q, want %q", result, "あ")
	}
}

func TestReadLineFromChannel_BackspaceUTF8MultibyteDelete(t *testing.T) {
	r, ch, _ := newTestReaderWithChannel()

	// Type "aあ" then backspace → should delete "あ" (3 bytes), leaving "a"
	// あ = E3 81 82
	go feedBytes(ch, []byte{'a', 0xE3, 0x81, 0x82, 0x7f})

	result, err := r.readLineFromChannel()
	if err != nil {
		t.Fatalf("readLineFromChannel() error = %v", err)
	}
	if result != "a" {
		t.Errorf("readLineFromChannel() = %q, want %q", result, "a")
	}
}

func TestReadLineFromChannel_BackspaceDEL(t *testing.T) {
	r, ch, _ := newTestReaderWithChannel()

	// 0x08 (BS) should also work as backspace
	go feedBytes(ch, []byte{'a', 'b', 0x08})

	result, err := r.readLineFromChannel()
	if err != nil {
		t.Fatalf("readLineFromChannel() error = %v", err)
	}
	if result != "a" {
		t.Errorf("readLineFromChannel() = %q, want %q", result, "a")
	}
}

func TestReadLineFromChannel_BackspaceMixed(t *testing.T) {
	r, ch, _ := newTestReaderWithChannel()

	// Type "hあiう" then 2 backspaces → "hあ"
	// あ = E3 81 82, う = E3 81 86
	go feedBytes(ch, []byte{
		'h',
		0xE3, 0x81, 0x82, // あ
		'i',
		0xE3, 0x81, 0x86, // う
		0x7f, // delete う
		0x7f, // delete i
	})

	result, err := r.readLineFromChannel()
	if err != nil {
		t.Fatalf("readLineFromChannel() error = %v", err)
	}
	expected := "hあ"
	if result != expected {
		t.Errorf("readLineFromChannel() = %q, want %q", result, expected)
	}
}
