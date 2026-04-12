package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// verifyViewLines checks total line count and each line width.
// Returns the split lines for further inspection.
func verifyViewLines(t *testing.T, m Model, label string) []string {
	t.Helper()
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) != m.height {
		t.Errorf("[%s] line count = %d, want %d (vp.height=%d, footer=%d)",
			label, len(lines), m.height, m.vp.height, m.footerHeight())
	}
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != m.width {
			t.Errorf("[%s] line %d: width = %d, want %d", label, i, w, m.width)
		}
	}
	return lines
}

// verifyFooterPosition checks that the footer (input dock + status bar) occupies
// exactly the last footerHeight() lines of View(). If the footer shifts up,
// transcript content would leak into the footer zone.
func verifyFooterPosition(t *testing.T, m Model, label string) {
	t.Helper()
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) != m.height {
		t.Errorf("[%s] line count = %d, want %d", label, len(lines), m.height)
		return
	}

	footerStart := m.height - m.footerHeight()

	// The viewport zone (lines 0..footerStart-1) should NOT contain input dock bg
	// patterns from chromeCache. The footer zone should contain them.
	// Check: the footer start line is a padLine (gray bg, all spaces after strip)
	padLine := lines[footerStart]
	padPlain := strings.TrimRight(stripANSI(padLine), " ")
	if padPlain != "" {
		t.Errorf("[%s] footer start line (line %d) has non-space content: %q",
			label, footerStart, padPlain)
	}

	// Check: input line contains the prompt
	inputLine := lines[footerStart+1]
	if !strings.Contains(inputLine, inputPrompt) && !strings.Contains(inputLine, "Type your message") {
		t.Errorf("[%s] footer input line (line %d) missing prompt/placeholder: %q",
			label, footerStart+1, inputLine)
	}
}

func setupBottomTestModel(lineCount int) Model {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 60
	m.height = 20
	vph := m.height - m.footerHeight()
	m.vp = lightViewport{width: 60, height: vph}
	m.textInput.Width = m.width - inputPromptWidth - 1
	m.padLineCache = fillANSITextWidth("", m.width, "\033[48;5;236m")
	for i := range lineCount {
		m.appendContentLines(fmt.Sprintf("Line %03d: %s", i, strings.Repeat("content ", 3)))
	}
	m.rebuildChrome()
	return m
}
