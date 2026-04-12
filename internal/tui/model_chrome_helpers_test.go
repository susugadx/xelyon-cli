package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// verifyViewStructure checks that View() output has exactly m.height lines,
// each with exactly m.width display columns.
func verifyViewStructure(t *testing.T, m Model, context string) {
	t.Helper()
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) != m.height {
		t.Errorf("[%s] View line count = %d, want %d", context, len(lines), m.height)
	}
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != m.width {
			t.Errorf("[%s] line %d: width = %d, want %d (line=%q)", context, i, w, m.width, line)
		}
	}
}

func setupModelForChromeTest(agent *stubAgent) Model {
	m := NewModel(agent, "")
	m.ready = true
	m.width = 80
	m.height = 24
	vph := m.height - m.footerHeight()
	m.vp = lightViewport{width: 80, height: vph}
	m.textInput.Width = m.width - inputPromptWidth - 1
	m.padLineCache = fillANSITextWidth("", m.width, "\033[48;5;236m")
	for i := range 30 {
		m.appendContentLines(strings.Repeat("x", 40) + string(rune('A'+i%26)))
	}
	m.rebuildChrome()
	return m
}
