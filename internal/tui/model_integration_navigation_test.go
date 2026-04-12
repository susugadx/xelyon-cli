package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIIntegration_NavigationVisualWheelResizeSequence(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 8, Height: 8})
	m = updated.(Model)
	m.navigationMode = true
	m.rawLines = []string{
		"alpha beta gamma",
		"second line",
		"third line",
		"fourth line",
		"fifth line",
		"sixth line",
	}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorLine = 0
	m.cursorCol = 0

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
	m = updated.(Model)
	updated, _ = m.Update(tea.WindowSizeMsg{Width: 14, Height: 10})
	m = updated.(Model)

	if m.visualMode != visualModeChar {
		t.Fatalf("visualMode = %d, want %d", m.visualMode, visualModeChar)
	}
	if m.visualStart != (visualPosition{line: 0, col: 6}) {
		t.Fatalf("visualStart = %+v, want {line:0 col:6}", m.visualStart)
	}
	if m.cursorLine != 1 {
		t.Fatalf("cursorLine = %d, want 1", m.cursorLine)
	}
	if m.cursorCol != 6 {
		t.Fatalf("cursorCol = %d, want 6", m.cursorCol)
	}
	view := m.View()
	if !strings.Contains(view, "\033[48;5;240m") {
		t.Fatalf("view should retain visual highlight after mixed events, got %q", view)
	}
}

func TestTUIIntegration_InputToNavigationLineCopyFlow(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 20, Height: 8})
	m = updated.(Model)
	m.rawLines = []string{"alpha", "beta", "gamma"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if !m.navigationMode {
		t.Fatal("Esc should enter navigation mode")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)

	if len(agent.copyTexts) != 1 {
		t.Fatalf("copyTexts len = %d, want 1", len(agent.copyTexts))
	}
	if agent.copyTexts[0] != "alpha\nbeta" {
		t.Fatalf("copied text = %q, want %q", agent.copyTexts[0], "alpha\nbeta")
	}
	if m.visualMode != visualModeOff {
		t.Fatalf("visualMode after copy = %d, want %d", m.visualMode, visualModeOff)
	}
	if !m.navigationMode {
		t.Fatal("navigationMode should remain true after copy")
	}
	if got := m.transientStatus; got != "✅ Copied 2 lines" {
		t.Fatalf("transientStatus = %q, want %q", got, "✅ Copied 2 lines")
	}
}
