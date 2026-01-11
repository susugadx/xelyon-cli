package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// MultilineReader handles multiline input with bracketed paste mode and ``` markers
type MultilineReader struct {
	reader          *bufio.Reader
	bracketedPasteEnabled bool
}

// NewMultilineReader creates a new multiline reader
func NewMultilineReader(r io.Reader) *MultilineReader {
	return &MultilineReader{
		reader:          bufio.NewReader(r),
		bracketedPasteEnabled: false,
	}
}

// EnableBracketedPaste enables bracketed paste mode (call once at start)
func (m *MultilineReader) EnableBracketedPaste() {
	if !m.bracketedPasteEnabled && isTerminal() {
		// CSI ? 2004 h - Enable bracketed paste mode
		fmt.Print("\x1b[?2004h")
		m.bracketedPasteEnabled = true
	}
}

// DisableBracketedPaste disables bracketed paste mode (call at cleanup)
func (m *MultilineReader) DisableBracketedPaste() {
	if m.bracketedPasteEnabled {
		// CSI ? 2004 l - Disable bracketed paste mode
		fmt.Print("\x1b[?2004l")
		m.bracketedPasteEnabled = false
	}
}

// ReadInput reads user input, supporting:
// 1. Bracketed paste mode (automatic detection)
// 2. ``` markers for explicit multiline mode
// 3. Single line input (default)
func (m *MultilineReader) ReadInput(prompt string) (string, error) {
	fmt.Print(prompt)

	// Read first line
	line, err := m.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	line = strings.TrimRight(line, "\n\r")

	// Case 1: Bracketed paste detected
	// Check for both \x1b[200~ (ESC [ 2 0 0 ~) and ^[[200~ (visual representation)
	if strings.HasPrefix(line, "\x1b[200~") || strings.Contains(line, "\x1b[200~") {
		return m.readBracketedPaste(line)
	}

	// Case 2: ``` marker detected - explicit multiline mode
	if line == "```" {
		return m.readMultilineWithMarker()
	}

	// Case 3: Single line input
	return line, nil
}

// readBracketedPaste handles bracketed paste mode input
func (m *MultilineReader) readBracketedPaste(firstLine string) (string, error) {
	// Remove bracketed paste start marker
	content := strings.TrimPrefix(firstLine, "\x1b[200~")

	// Check if end marker is already in the first line (single-line paste)
	if strings.Contains(content, "\x1b[201~") {
		result := strings.TrimSuffix(content, "\x1b[201~")
		lineCount := strings.Count(result, "\n") + 1
		if result == "" {
			lineCount = 0
		}
		fmt.Printf("📋 Pasted %d lines\n", lineCount)
		return result, nil
	}

	var lines []string
	if content != "" {
		lines = append(lines, content)
	}

	// Read until we find the end marker
	for {
		line, err := m.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}

		line = strings.TrimRight(line, "\n\r")

		// Check for end marker
		if strings.Contains(line, "\x1b[201~") {
			line = strings.TrimSuffix(line, "\x1b[201~")
			if line != "" {
				lines = append(lines, line)
			}
			break
		}

		lines = append(lines, line)
	}

	result := strings.Join(lines, "\n")
	fmt.Printf("📋 Pasted %d lines\n", len(lines))
	return result, nil
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

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// IsMultilineMarker checks if the input is a multiline marker
func IsMultilineMarker(input string) bool {
	return input == "```"
}

// TrimBracketedPasteMarkers removes bracketed paste markers from input
func TrimBracketedPasteMarkers(input string) string {
	input = strings.TrimPrefix(input, "\x1b[200~")
	input = strings.TrimSuffix(input, "\x1b[201~")
	return input
}
