package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNavigationMotionInput_ZeroExtendsPendingCount(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.navigationMode = true
	setModelRawLines(&m, 40)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	m = updated.(Model)

	if m.pendingCount != 10 {
		t.Fatalf("pendingCount = %d, want 10", m.pendingCount)
	}
	if m.cursorLine != 0 {
		t.Fatalf("cursorLine after count prefix = %d, want 0", m.cursorLine)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	if m.cursorLine != 10 {
		t.Fatalf("cursorLine after 10j = %d, want 10", m.cursorLine)
	}
	if m.pendingCount != 0 {
		t.Fatalf("pendingCount after 10j = %d, want 0", m.pendingCount)
	}
}

func TestNavigationCopyInput_YCopiesFocusedBlockDetail(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.navigationMode = true

	m.appendToolResult(ToolResult{
		Name:      "read_file",
		Summary:   "read_file: sample.txt",
		Detail:    "first line\nsecond line",
		Collapsed: true,
	})
	m.setBlockFocus(0)

	updated, _ := m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)

	if agent.copyCalls != 1 {
		t.Fatalf("copyCalls = %d, want 1", agent.copyCalls)
	}
	if len(agent.copyTexts) != 1 || agent.copyTexts[0] != "first line\nsecond line" {
		t.Fatalf("copyTexts = %#v, want focused block detail", agent.copyTexts)
	}
	if m.transientStatus == "" {
		t.Fatal("transientStatus should be set after focused block copy")
	}
	if m.focusedBlock != 0 {
		t.Fatalf("focusedBlock = %d, want 0", m.focusedBlock)
	}
}
