package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newProjectTestModel(agent *stubAgent) Model {
	m := newModelWithViewport(agent)
	m.screen = screenProject
	m.installProjectScreen(agent.projectConfig)
	m.projectScreen.normalizeSize(m.width, m.height)
	m.rebuildChrome()
	return m
}

func sendProjectKey(m Model, s string) Model {
	var msg tea.KeyMsg
	switch s {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	default:
		if len(s) == 1 {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
		}
	}
	updated, _ := m.Update(msg)
	return updated.(Model)
}

func sendProjectCtrlS(m Model) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	return updated.(Model)
}

func editProjectContext(t *testing.T, m Model, value string) Model {
	t.Helper()
	m = sendProjectKey(m, "enter")
	if m.projectScreen.editMode != projectEditContext {
		t.Fatalf("editMode = %d, want projectEditContext(%d)", m.projectScreen.editMode, projectEditContext)
	}
	m.projectScreen.contextArea.SetValue(value)
	return sendProjectCtrlS(m)
}

func saveProjectAndWait(t *testing.T, m Model) Model {
	t.Helper()
	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}
	updated, _ = m.Update(saveCmd())
	return updated.(Model)
}
