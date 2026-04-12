package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNavMode_JKScrolls(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 5}
	setModelRawLines(&m, 20)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	got := updated.(Model)
	if got.cursorLine != 1 {
		t.Fatalf("j: cursorLine = %d, want 1", got.cursorLine)
	}

	updated, _ = got.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	got = updated.(Model)
	if got.cursorLine != 0 {
		t.Fatalf("k: cursorLine = %d, want 0", got.cursorLine)
	}
}

func TestNavMode_DUHalfPage(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 10}
	setModelRawLines(&m, 40)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	got := updated.(Model)
	if got.cursorLine != 5 {
		t.Fatalf("d: cursorLine = %d, want 5", got.cursorLine)
	}

	updated, _ = got.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	got = updated.(Model)
	if got.cursorLine != 0 {
		t.Fatalf("u: cursorLine = %d, want 0", got.cursorLine)
	}
}

func TestNavMode_VerticalMoveClampsCursorToVisibleWidth(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.width = 5
	m.height = 8
	m.vp = lightViewport{width: 5, height: 4}
	m.rawLines = []string{"abcdefghij", "klmnopqrst"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorLine = 0
	m.cursorCol = 4

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	if m.cursorLine != 1 {
		t.Fatalf("cursorLine = %d, want 1", m.cursorLine)
	}
	if m.cursorCol != 4 {
		t.Fatalf("cursorCol = %d, want 4", m.cursorCol)
	}

	m.cursorCol = 8
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(Model)

	if m.cursorLine != 0 {
		t.Fatalf("cursorLine after k = %d, want 0", m.cursorLine)
	}
	if m.cursorCol != 8 {
		t.Fatalf("cursorCol after k = %d, want 8", m.cursorCol)
	}

	m.rebuildChrome()
	view := m.View()
	if !strings.Contains(view, "\033[48;5;255;38;5;16mi") {
		t.Fatalf("view should contain visible cursor after vertical move, got %q", view)
	}
}

func TestNavMode_MovingToBottomClearsNewOutputBadge(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.width = 80
	m.height = 8
	m.vp = lightViewport{width: 80, height: 4}

	for i := 0; i < 20; i++ {
		m.appendContentLines(fmt.Sprintf("line-%d", i))
	}
	m.vp.gotoTop()
	m.appendContentLines("new-line")
	m.chromeDirty = true
	m.rebuildChrome()

	if !m.newOutput {
		t.Fatal("newOutput should be true while scrolled away from bottom")
	}
	if !strings.Contains(m.chromeCache, "New output") {
		t.Fatalf("chromeCache should include new output badge, got %q", m.chromeCache)
	}

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	m.rebuildChrome()

	if !m.vp.atBottom() {
		t.Fatal("viewport should be at bottom after G")
	}
	if m.newOutput {
		t.Fatal("newOutput should be cleared when cursor move reaches bottom")
	}
	if strings.Contains(m.chromeCache, "New output") {
		t.Fatalf("chromeCache should clear new output badge, got %q", m.chromeCache)
	}
}
