package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNavMode_VisualSelectionCopiesPlainText(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 20
	m.height = 8
	m.navigationMode = true
	m.vp = lightViewport{width: 20, height: 5}
	m.rawLines = []string{"one", "\033[31m二行目\033[0m", "three"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	m = updated.(Model)
	if m.visualMode != visualModeLine {
		t.Fatalf("visualMode = %d, want %d", m.visualMode, visualModeLine)
	}
	if m.visualStart.line != 0 {
		t.Fatalf("visualStart.line = %d, want 0", m.visualStart.line)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	if m.cursorLine != 1 {
		t.Fatalf("cursorLine = %d, want 1", m.cursorLine)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)
	if m.visualMode != visualModeOff {
		t.Fatalf("visualMode after copy = %d, want %d", m.visualMode, visualModeOff)
	}
	if len(agent.copyTexts) != 1 {
		t.Fatalf("copyTexts len = %d, want 1", len(agent.copyTexts))
	}
	if agent.copyTexts[0] != "one\n二行目" {
		t.Fatalf("copied text = %q, want %q", agent.copyTexts[0], "one\n二行目")
	}
}

func TestNavMode_CharVisualSelectionCopiesRange(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 20, height: 5}
	m.rawLines = []string{"hello", "world"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorLine = 0
	m.cursorCol = 1

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	if m.visualMode != visualModeChar {
		t.Fatalf("visualMode = %d, want %d", m.visualMode, visualModeChar)
	}
	if m.visualStart.col != 1 {
		t.Fatalf("visualStart.col = %d, want 1", m.visualStart.col)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)

	if len(agent.copyTexts) != 1 {
		t.Fatalf("copyTexts len = %d, want 1", len(agent.copyTexts))
	}
	if agent.copyTexts[0] != "ell" {
		t.Fatalf("copied text = %q, want %q", agent.copyTexts[0], "ell")
	}
	if m.visualMode != visualModeOff {
		t.Fatalf("visualMode after copy = %d, want %d", m.visualMode, visualModeOff)
	}
}

func TestNavMode_CharVisualSelectionCopiesAcrossTabDisplayWidth(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 20, height: 5}
	m.rawLines = []string{"a\tb"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	for range 5 {
		updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
		m = updated.(Model)
	}
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)

	if len(agent.copyTexts) != 1 {
		t.Fatalf("copyTexts len = %d, want 1", len(agent.copyTexts))
	}
	if agent.copyTexts[0] != "a\tb" {
		t.Fatalf("copied text = %q, want %q", agent.copyTexts[0], "a\\tb")
	}
	if m.visualMode != visualModeOff {
		t.Fatalf("visualMode after copy = %d, want %d", m.visualMode, visualModeOff)
	}
}

func TestNavMode_WordMotionsWorkInVisualMode(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 30, height: 5}
	m.rawLines = []string{"alpha beta gamma"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)

	if len(agent.copyTexts) != 1 {
		t.Fatalf("copyTexts len = %d, want 1", len(agent.copyTexts))
	}
	if agent.copyTexts[0] != "alpha beta" {
		t.Fatalf("copied text = %q, want %q", agent.copyTexts[0], "alpha beta")
	}
}
