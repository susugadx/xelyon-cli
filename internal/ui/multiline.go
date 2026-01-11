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
	reader                *bufio.Reader
	bracketedPasteEnabled bool
}

// NewMultilineReader creates a new multiline reader
func NewMultilineReader(r io.Reader) *MultilineReader {
	return &MultilineReader{
		reader:                bufio.NewReader(r),
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

// Bracketed paste markers - both ESC sequence and literal ^[ forms
const (
	bracketedPasteStart    = "\x1b[200~"
	bracketedPasteEnd      = "\x1b[201~"
	bracketedPasteStartAlt = "^[[200~" // Literal form (some terminals)
	bracketedPasteEndAlt   = "^[[201~" // Literal form (some terminals)
)

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

	// Case 1: Bracketed paste detected (check both ESC and literal ^[ forms)
	if hasBracketedPasteStart(line) {
		return m.readBracketedPaste(line)
	}

	// Case 2: ``` marker detected - explicit multiline mode
	if line == "```" {
		return m.readMultilineWithMarker()
	}

	// Case 3: Single line input
	return line, nil
}

// hasBracketedPasteStart checks if line contains bracketed paste start marker
func hasBracketedPasteStart(line string) bool {
	return strings.Contains(line, bracketedPasteStart) ||
		strings.Contains(line, bracketedPasteStartAlt)
}

// hasBracketedPasteEnd checks if line contains bracketed paste end marker
func hasBracketedPasteEnd(line string) bool {
	return strings.Contains(line, bracketedPasteEnd) ||
		strings.Contains(line, bracketedPasteEndAlt)
}

// removeBracketedPasteStart removes the start marker from the line
func removeBracketedPasteStart(line string) string {
	line = strings.Replace(line, bracketedPasteStart, "", 1)
	line = strings.Replace(line, bracketedPasteStartAlt, "", 1)
	return line
}

// removeBracketedPasteEnd removes the end marker from the line
func removeBracketedPasteEnd(line string) string {
	line = strings.Replace(line, bracketedPasteEnd, "", 1)
	line = strings.Replace(line, bracketedPasteEndAlt, "", 1)
	return line
}

// readBracketedPaste handles bracketed paste mode input
func (m *MultilineReader) readBracketedPaste(firstLine string) (string, error) {
	// Remove bracketed paste start marker (both ESC and literal forms)
	content := removeBracketedPasteStart(firstLine)

	// Check if end marker is already in the first line (single-line paste)
	if hasBracketedPasteEnd(content) {
		result := removeBracketedPasteEnd(content)
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

		// Check for end marker (both ESC and literal forms)
		if hasBracketedPasteEnd(line) {
			line = removeBracketedPasteEnd(line)
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

// TrimBracketedPasteMarkers removes bracketed paste markers from input (both forms)
func TrimBracketedPasteMarkers(input string) string {
	input = removeBracketedPasteStart(input)
	input = removeBracketedPasteEnd(input)
	return input
}
