package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNavMode_DollarMovesToVisualEndOnTabExpandedLine(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 20
	m.height = 8
	m.navigationMode = true
	m.vp = lightViewport{width: 20, height: 5}
	m.rawLines = []string{"a\tb"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'$'}})
	m = updated.(Model)

	if m.cursorCol != 5 {
		t.Fatalf("cursorCol = %d, want 5", m.cursorCol)
	}
	view := m.View()
	if !strings.Contains(view, "\033[48;5;255;38;5;16mb") {
		t.Fatalf("view should place cursor on trailing rune after tab expansion, got %q", view)
	}
}

func TestNavMode_HLMoveCursorColWithoutVisualMode(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 20, height: 5}
	m.rawLines = []string{"hello"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updated.(Model)

	if m.cursorCol != 1 {
		t.Fatalf("cursorCol = %d, want 1", m.cursorCol)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	if m.visualStart.col != 1 {
		t.Fatalf("visualStart.col = %d, want 1", m.visualStart.col)
	}
}

func TestNavMode_ZeroMovesToLineStartWithoutCount(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 20, height: 5}
	m.rawLines = []string{"hello world"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorCol = 5

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	m = updated.(Model)
	if m.cursorCol != 0 {
		t.Fatalf("cursorCol = %d, want 0", m.cursorCol)
	}
}

func TestNavMode_WBEWordMotions(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 30, height: 5}
	m.rawLines = []string{"alpha beta gamma"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = updated.(Model)
	if m.cursorCol != 6 {
		t.Fatalf("cursorCol after w = %d, want 6", m.cursorCol)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = updated.(Model)
	if m.cursorCol != 9 {
		t.Fatalf("cursorCol after e = %d, want 9", m.cursorCol)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = updated.(Model)
	if m.cursorCol != 6 {
		t.Fatalf("cursorCol after b = %d, want 6", m.cursorCol)
	}
}

func TestNavMode_LineStartAndEndMotions(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 30, height: 5}
	m.rawLines = []string{"  alpha", " beta", "gamma"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'^'}})
	m = updated.(Model)
	if m.cursorCol != 2 {
		t.Fatalf("cursorCol after ^ = %d, want 2", m.cursorCol)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'$'}})
	m = updated.(Model)
	if m.cursorCol != 6 {
		t.Fatalf("cursorCol after $ = %d, want 6", m.cursorCol)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'$'}})
	m = updated.(Model)
	if m.cursorLine != 1 {
		t.Fatalf("cursorLine after 2$ = %d, want 1", m.cursorLine)
	}
	if m.cursorCol != 4 {
		t.Fatalf("cursorCol after 2$ = %d, want 4", m.cursorCol)
	}
}

func TestNavMode_WordMotionKeepsLogicalColumnOnLongLine(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.width = 5
	m.height = 8
	m.vp = lightViewport{width: 5, height: 4}
	m.rawLines = []string{"abcde fg", "tail"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorLine = 0
	m.cursorCol = 4
	m.rebuildChrome()

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = updated.(Model)

	if m.cursorLine != 0 {
		t.Fatalf("cursorLine after w = %d, want 0", m.cursorLine)
	}
	if m.cursorCol != 6 {
		t.Fatalf("cursorCol after w = %d, want 6", m.cursorCol)
	}

	view := m.View()
	if !strings.Contains(view, "\033[48;5;255;38;5;16mf") {
		t.Fatalf("view should keep cursor visible at edge for off-screen column, got %q", view)
	}
}
