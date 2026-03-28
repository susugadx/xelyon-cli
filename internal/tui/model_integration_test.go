package tui

import (
	"fmt"
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

func TestTUIIntegration_ToolBlockFocusExpandWheelCollapseSequence(t *testing.T) {
	m := NewModel(&stubAgent{statusLine: "ready"}, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 10, Height: 9})
	m = updated.(Model)
	m.navigationMode = true
	for i := 0; i < 12; i++ {
		m.appendContentLines(fmt.Sprintf("pad-%d", i))
	}
	m.appendToolResult(ToolResult{
		Name:      "bash",
		Summary:   "bash",
		Detail:    "\033[32m0123456789abcdef\033[0m",
		Collapsed: true,
	})

	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	updated, _ = m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress})
	m = updated.(Model)
	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.focusedBlock != 0 {
		t.Fatalf("focusedBlock = %d, want 0", m.focusedBlock)
	}
	if !m.toolBlocks[0].tool.Collapsed {
		t.Fatal("block should be collapsed again after second Enter")
	}
	if m.toolBlocks[0].lineStart <= m.toolBlocks[0].lineCount-1 {
		t.Fatalf("unexpected tool block lineStart = %d", m.toolBlocks[0].lineStart)
	}
	if m.vp.yOffset < 0 {
		t.Fatalf("viewport yOffset = %d, want >= 0", m.vp.yOffset)
	}
}
