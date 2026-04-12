package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestComposer_EscDoesNotEnterNavigationWithFoldedPaste(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	updated, _ := m.Update(pasteKey("line1\nline2"))
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.navigationMode {
		t.Fatal("navigationMode should stay false while composer has folded paste blocks")
	}
}

func TestComposer_EscTreatsWhitespaceOnlyInputAsEmpty(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.textInput.SetValue(" \t ")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if !m.navigationMode {
		t.Fatal("navigationMode should become true for whitespace-only input")
	}
}
