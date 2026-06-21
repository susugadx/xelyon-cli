package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/projectscreen"
)

func newProjectTestModel(agent *stubAgent) Model {
	m := newModelWithViewport(agent)
	m.screen = screenProject
	m.installProjectScreen(agent.projectConfig)
	m.projectScreen.NormalizeSize(m.width, m.height)
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
	if got := projectSnapshot(t, m).EditMode; got != "context" {
		t.Fatalf("editMode = %s, want context", got)
	}
	m = sendProjectText(m, value)
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

func projectSnapshot(t *testing.T, m Model) projectscreen.Snapshot {
	t.Helper()
	if m.projectScreen == nil {
		t.Fatal("projectScreen is nil")
	}
	return m.projectScreen.Snapshot()
}

func sendProjectText(m Model, value string) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value), Paste: true})
	return updated.(Model)
}

func moveProjectToSection(t *testing.T, m Model, section string) Model {
	t.Helper()
	for i := 0; i < 12; i++ {
		if projectSnapshot(t, m).Section == section {
			return m
		}
		m = sendProjectKey(m, "down")
	}
	t.Fatalf("project section did not reach %q; got %q", section, projectSnapshot(t, m).Section)
	return m
}

func moveProjectToItemPane(t *testing.T, m Model) Model {
	t.Helper()
	m = sendProjectKey(m, "enter")
	if got := projectSnapshot(t, m).ActivePane; got != "item" {
		t.Fatalf("activePane = %s, want item", got)
	}
	return m
}

func moveProjectItemSelection(t *testing.T, m Model, index int) Model {
	t.Helper()
	for i := 0; i < index; i++ {
		m = sendProjectKey(m, "down")
	}
	if got := projectSnapshot(t, m).SelectedIndex; got != index {
		t.Fatalf("selected index = %d, want %d", got, index)
	}
	return m
}
