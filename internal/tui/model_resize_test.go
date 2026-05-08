package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModel_WindowResizeRestoresFullLineFromRawContent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 5, Height: 8})
	m = updated.(Model)

	updated, _ = m.Update(AppendMessageMsg{
		Message: ChatMessage{Role: "assistant", Content: "123456789"},
	})
	m = updated.(Model)

	if got := stripANSI(m.getVisualRowContents()[len(m.getVisualRowContents())-1]); got != "9" {
		t.Fatalf("narrow render last row = %q, want %q", got, "9")
	}

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 12, Height: 8})
	m = updated.(Model)

	if got := stripANSI(m.rawLines[len(m.rawLines)-1]); got != "│ 123456789" {
		t.Fatalf("raw line plain = %q, want %q", got, "│ 123456789")
	}
	if got := stripANSI(m.getVisualRowContents()[len(m.getVisualRowContents())-1]); got != "│ 123456789" {
		t.Fatalf("wide render plain = %q, want %q", got, "│ 123456789")
	}
}

func TestModel_WindowResizeKeepsCharVisualSelectionState(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 6, Height: 8})
	m = updated.(Model)
	m.navigationMode = true
	m.rawLines = []string{"abcdefghij"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorLine = 0
	m.cursorCol = 1

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	m.cursorCol = 7
	m.rebuildChrome()

	before := m.View()
	if !strings.Contains(before, "\033[48;5;255;38;5;16mh") {
		t.Fatalf("narrow view should keep cursor visible before resize, got %q", before)
	}

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 12, Height: 8})
	m = updated.(Model)
	after := m.View()

	if m.visualMode != visualModeChar {
		t.Fatalf("visualMode = %d, want %d", m.visualMode, visualModeChar)
	}
	if m.visualStart != (visualPosition{line: 0, col: 1}) {
		t.Fatalf("visualStart = %+v, want {line:0 col:1}", m.visualStart)
	}
	if m.cursorCol != 7 {
		t.Fatalf("cursorCol = %d, want 7", m.cursorCol)
	}
	if !strings.Contains(stripANSI(after), "abcdefgh") {
		t.Fatalf("wide view should restore full visible selection context, got %q", after)
	}
}

func TestModel_WindowResizeKeepsLineVisualSelectionState(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 8, Height: 8})
	m = updated.(Model)
	m.navigationMode = true
	setModelRawLines(&m, 20)
	m.cursorLine = 2

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 20, Height: 10})
	m = updated.(Model)

	if m.visualMode != visualModeLine {
		t.Fatalf("visualMode = %d, want %d", m.visualMode, visualModeLine)
	}
	if m.visualStart.line != 2 {
		t.Fatalf("visualStart.line = %d, want 2", m.visualStart.line)
	}
	if m.cursorLine != 4 {
		t.Fatalf("cursorLine = %d, want 4", m.cursorLine)
	}
	view := m.View()
	if !strings.Contains(view, "\033[48;5;240m") {
		t.Fatalf("line visual selection should remain highlighted after resize, got %q", view)
	}
}
