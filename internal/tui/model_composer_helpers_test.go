package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func pasteKey(text string) tea.KeyMsg {
	return tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(text),
		Paste: true,
	}
}

func stubClipboardRead(t *testing.T, text string, err error) {
	t.Helper()
	prev := readClipboardText
	readClipboardText = func() (string, error) {
		return text, err
	}
	t.Cleanup(func() {
		readClipboardText = prev
	})
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
