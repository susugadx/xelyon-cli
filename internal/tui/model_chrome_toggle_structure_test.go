package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestView_EscToggle_StructureConsistent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	for i := range 10 {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = updated.(Model)
		verifyViewStructure(t, m, "ESC press "+string(rune('0'+i)))
	}
}

func TestView_EscToggle_PlaceholderNotDuplicated(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	for range 6 {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = updated.(Model)
	}

	view := m.View()
	count := strings.Count(view, "Type your message")
	if count > 1 {
		t.Errorf("placeholder appears %d times, want <=1", count)
	}
}

func TestView_EscToggle_StatusBarNotDuplicated(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	for range 6 {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = updated.(Model)
	}

	view := m.View()
	badgeCount := strings.Count(view, "\033[48;5;33;38;5;255m NAV \033[0m")
	if badgeCount > 1 {
		t.Errorf("NAV badge appears %d times, want <=1", badgeCount)
	}
}
