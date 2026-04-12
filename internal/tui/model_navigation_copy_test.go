package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNavMode_YCallsCopy(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 5}
	setModelRawLines(&m, 5)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	got := updated.(Model)
	if agent.copyCalls != 0 {
		t.Fatalf("first y should wait for yy, copyCalls = %d", agent.copyCalls)
	}
	if !got.yPressed {
		t.Fatal("first y should set yPressed")
	}

	updated, _ = got.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	got = updated.(Model)
	if agent.copyCalls != 1 {
		t.Fatalf("copyCalls = %d, want 1", agent.copyCalls)
	}
	if len(agent.copyTexts) != 1 || agent.copyTexts[0] != "line0" {
		t.Fatalf("copied text = %#v, want [line0]", agent.copyTexts)
	}
	if got.transientStatus == "" {
		t.Fatal("transientStatus should be set after copy")
	}
}

func TestNavMode_PendingYFallsBackToCopyLastOutput(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 5}
	setModelRawLines(&m, 5)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	if agent.copyCalls != 1 {
		t.Fatalf("copyCalls = %d, want 1", agent.copyCalls)
	}
	if m.cursorLine != 1 {
		t.Fatalf("cursorLine = %d, want 1", m.cursorLine)
	}
}
