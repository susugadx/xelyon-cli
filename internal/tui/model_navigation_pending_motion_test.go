package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNavMode_CountPrefixAppliesToMoves(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 10}
	setModelRawLines(&m, 40)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = updated.(Model)
	if m.pendingCount != 3 {
		t.Fatalf("pendingCount = %d, want 3", m.pendingCount)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	if m.cursorLine != 3 {
		t.Fatalf("cursorLine = %d, want 3", m.cursorLine)
	}
	if m.pendingCount != 0 {
		t.Fatalf("pendingCount = %d, want 0", m.pendingCount)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(Model)
	if m.cursorLine != 13 {
		t.Fatalf("cursorLine after 2d = %d, want 13", m.cursorLine)
	}
}

func TestNavMode_GGAndG(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 5}
	setModelRawLines(&m, 20)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	got := updated.(Model)
	if got.cursorLine != 19 {
		t.Fatalf("G: cursorLine = %d, want 19", got.cursorLine)
	}

	updated, _ = got.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	got = updated.(Model)
	if !got.gPressed {
		t.Fatal("first g should set gPressed")
	}
	updated, _ = got.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	got = updated.(Model)
	if got.cursorLine != 0 {
		t.Fatalf("gg: cursorLine = %d, want 0", got.cursorLine)
	}
	if got.gPressed {
		t.Fatal("gPressed should be reset after gg")
	}
}

func TestNavMode_CountPrefixAppliesToGGAndG(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 5}
	setModelRawLines(&m, 20)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	if m.cursorLine != 11 {
		t.Fatalf("cursorLine after 12G = %d, want 11", m.cursorLine)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(Model)
	if m.cursorLine != 2 {
		t.Fatalf("cursorLine after 3gg = %d, want 2", m.cursorLine)
	}
}

func TestNavMode_GFollowedByOtherKeyResetsG(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.gPressed = true

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	got := updated.(Model)
	if got.gPressed {
		t.Fatal("gPressed should be reset after non-g key")
	}
}

func TestNavMode_CountCanIncludeZero(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 5}
	setModelRawLines(&m, 20)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	if m.cursorLine != 10 {
		t.Fatalf("cursorLine = %d, want 10", m.cursorLine)
	}
}
