package ui

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// MultilineReader handles multiline input with bracketed paste mode and ``` markers
type MultilineReader struct {
	reader                *bufio.Reader
	bracketedPasteEnabled bool
}

// NewMultilineReader creates a new multiline reader
func NewMultilineReader(r io.Reader) *MultilineReader {
	return &MultilineReader{
		reader:                bufio.NewReaderSize(r, 1024*1024), // 1MB buffer
		bracketedPasteEnabled: false,
	}
}

// EnableBracketedPaste is a no-op (kept for API compatibility)
// Note: We don't enable bracketed paste mode because some terminals (WSL/Ubuntu)
// display the escape sequences as literal text. Instead, we just strip markers.
func (m *MultilineReader) EnableBracketedPaste() {
	// Intentionally disabled - markers are stripped in ReadInput
}

// DisableBracketedPaste is a no-op (kept for API compatibility)
func (m *MultilineReader) DisableBracketedPaste() {
	// Intentionally disabled
}

// stripAllBracketedPasteMarkers removes all bracketed paste markers from input
// Handles both ESC sequence forms (\x1b[200~) and literal forms (^[[200~)
func stripAllBracketedPasteMarkers(input string) string {
	// ESC sequence forms
	input = strings.ReplaceAll(input, "\x1b[200~", "")
	input = strings.ReplaceAll(input, "\x1b[201~", "")
	// Literal ^[ forms (displayed as text in some terminals)
	input = strings.ReplaceAll(input, "^[[200~", "")
	input = strings.ReplaceAll(input, "^[[201~", "")
	return input
}

// ReadInput reads user input, supporting:
// 1. ``` markers for explicit multiline mode
// 2. Single line input (default)
// All bracketed paste markers are automatically stripped from input
func (m *MultilineReader) ReadInput(prompt string) (string, error) {
	fmt.Print(prompt)

	// Read first line
	line, err := m.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	line = strings.TrimRight(line, "\n\r")

	// Always strip bracketed paste markers
	line = stripAllBracketedPasteMarkers(line)

	// Case 1: ``` marker detected - explicit multiline mode
	if line == "```" {
		return m.readMultilineWithMarker()
	}

	// Case 2: Single line input
	return line, nil
}

// readMultilineWithMarker handles explicit multiline mode with ``` markers
func (m *MultilineReader) readMultilineWithMarker() (string, error) {
	fmt.Println("📝 Multiline input mode (end with ``` on a new line)")

	var lines []string
	lineNum := 1

	for {
		fmt.Printf("%3d | ", lineNum)
		line, err := m.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && len(lines) > 0 {
				break
			}
			return "", err
		}

		line = strings.TrimRight(line, "\n\r")

		// Check for end marker
		if line == "```" {
			break
		}

		lines = append(lines, line)
		lineNum++
	}

	result := strings.Join(lines, "\n")
	fmt.Printf("✅ Captured %d lines\n", len(lines))
	return result, nil
}

// IsMultilineMarker checks if the input is a multiline marker
func IsMultilineMarker(input string) bool {
	return input == "```"
}

// TrimBracketedPasteMarkers removes bracketed paste markers from input (both forms)
func TrimBracketedPasteMarkers(input string) string {
	return stripAllBracketedPasteMarkers(input)
}

// FlushInput discards any buffered input data
// This should be called after AI output completes to ignore keypresses during output
func (m *MultilineReader) FlushInput() {
	// Discard all buffered data
	if _, err := m.reader.Discard(m.reader.Buffered()); err != nil {
		// Best-effort: ignore flush errors to avoid breaking interactive UX
		return
	}
}
