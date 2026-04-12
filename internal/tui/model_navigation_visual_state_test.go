package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNavMode_EscCancelsVisualSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.visualMode = visualModeLine
	m.visualStart = visualPosition{line: 1, col: 0}
	m.cursorLine = 2

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(Model)
	if got.visualMode != visualModeOff {
		t.Fatalf("visualMode = %d, want %d", got.visualMode, visualModeOff)
	}
	if !got.navigationMode {
		t.Fatal("Esc in visual mode should stay in navigation mode")
	}
}

func TestNavMode_CharVisualSelectionSupportsLineMoveAndColumnClamp(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 20, height: 5}
	m.rawLines = []string{"abcdef", "xy"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorCol = 4

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	if m.cursorLine != 1 {
		t.Fatalf("cursorLine = %d, want 1", m.cursorLine)
	}
	if m.cursorCol != 1 {
		t.Fatalf("cursorCol = %d, want 1", m.cursorCol)
	}
}

func TestModel_MouseWheelDoesNotChangeVisualSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.vp = lightViewport{width: 20, height: 4}
	setModelRawLines(&m, 20)
	m.cursorLine = 2
	m.cursorCol = 0

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	startBefore := m.visualStart
	cursorLineBefore := m.cursorLine
	cursorColBefore := m.cursorCol
	yOffsetBefore := m.vp.yOffset

	updated, _ = m.Update(tea.MouseMsg{
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	m = updated.(Model)

	if m.vp.yOffset != yOffsetBefore+3 {
		t.Fatalf("yOffset after wheel = %d, want %d", m.vp.yOffset, yOffsetBefore+3)
	}
	if m.visualStart != startBefore {
		t.Fatalf("visualStart changed from %+v to %+v", startBefore, m.visualStart)
	}
	if m.cursorLine != cursorLineBefore {
		t.Fatalf("cursorLine changed from %d to %d during visual wheel scroll", cursorLineBefore, m.cursorLine)
	}
	if m.cursorCol != cursorColBefore {
		t.Fatalf("cursorCol changed from %d to %d during visual wheel scroll", cursorColBefore, m.cursorCol)
	}
}
