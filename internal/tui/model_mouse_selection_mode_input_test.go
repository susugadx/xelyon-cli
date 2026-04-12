package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestEsc_ClearsMouseSelection_InputMode(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.rebuildChrome()
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 2, col: 5}

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.hasActiveMouseSelection() {
		t.Fatal("expected selection cleared by Esc")
	}
	if m.navigationMode {
		t.Fatal("Esc should clear selection first, not enter nav mode")
	}
}

func TestEsc_ClearsMouseSelection_NavMode(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.rebuildChrome()
	m.navigationMode = true
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 2, col: 5}

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.hasActiveMouseSelection() {
		t.Fatal("expected selection cleared by Esc in nav mode")
	}
	if !m.navigationMode {
		t.Fatal("should stay in nav mode after clearing selection")
	}
}

func TestVisualV_ClearsMouseSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.rebuildChrome()
	m.navigationMode = true
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 2, col: 5}

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)

	if m.hasActiveMouseSelection() {
		t.Fatal("expected mouse selection cleared when entering visual mode")
	}
}

func TestVisualV_Line_ClearsMouseSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.rebuildChrome()
	m.navigationMode = true
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 2, col: 5}

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	m = updated.(Model)

	if m.hasActiveMouseSelection() {
		t.Fatal("expected mouse selection cleared when entering visual line mode")
	}
}
