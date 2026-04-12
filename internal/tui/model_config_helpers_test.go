package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func newConfigTestModel() Model {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)
	return m
}

func sendConfigKey(m Model, s string) Model {
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
	case "left":
		msg = tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		msg = tea.KeyMsg{Type: tea.KeyRight}
	default:
		if len(s) == 1 {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
		}
	}
	updated, _ := m.Update(msg)
	return updated.(Model)
}

func sendConfigKeys(m Model, keys ...string) Model {
	for _, k := range keys {
		m = sendConfigKey(m, k)
	}
	return m
}

func setConfigFieldSelection(t *testing.T, cs *configScreen, categoryName, fieldPath string) {
	t.Helper()

	for i, cat := range cs.categories {
		if cat.Name == categoryName {
			cs.catIndex = i
			cs.activePane = paneField
			for j, f := range cs.filteredFields() {
				if f.Path == fieldPath {
					cs.fieldIndex = j
					return
				}
			}
			t.Fatalf("field %q not found in category %q", fieldPath, categoryName)
		}
	}

	t.Fatalf("category %q not found", categoryName)
}

func selectConfigOption(t *testing.T, m Model, categoryName, fieldPath, option string) Model {
	t.Helper()

	cs := m.configScreen
	setConfigFieldSelection(t, cs, categoryName, fieldPath)

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editSelect {
		t.Fatalf("editMode = %d, want editSelect", cs.editMode)
	}

	field := cs.selectedField()
	if field == nil {
		t.Fatal("selectedField is nil")
	}

	found := false
	for i, candidate := range field.Options {
		if candidate == option {
			cs.editSelect = i
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("option %q not found in %s", option, fieldPath)
	}

	m = sendConfigKey(m, "enter")
	return m
}

func saveConfigAndWait(t *testing.T, m Model) Model {
	t.Helper()

	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}

	resultMsg := saveCmd()
	updated, _ = m.Update(resultMsg)
	return updated.(Model)
}

func enterStructMapEdit(t *testing.T, path string) Model {
	t.Helper()
	m := newConfigTestModel()
	cs := m.configScreen

	parts := strings.SplitN(path, ".", 2)
	catName := parts[0]
	if path == "provider_models" {
		catName = "provider"
	}
	for i, cat := range cs.categories {
		if cat.Name == catName {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == path {
			cs.fieldIndex = i
			break
		}
	}
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editStructMap {
		t.Fatalf("editMode = %d, want editStructMap", cs.editMode)
	}
	return m
}

func enterStructMapEntryForKey(t *testing.T, path, key string) Model {
	t.Helper()
	m := enterStructMapEdit(t, path)
	cs := m.configScreen

	for i, k := range cs.editStructKeys {
		if k == key {
			cs.editStructIndex = i
			break
		}
	}
	if cs.editStructKeys[cs.editStructIndex] != key {
		t.Fatalf("key %q not found in editStructKeys", key)
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if !cs.editEntryActive {
		t.Fatalf("editEntryActive should be true for key %q", key)
	}
	return m
}

func setEntryFieldIndex(t *testing.T, cs *configScreen, name string) int {
	t.Helper()
	for i, ef := range cs.editEntryFields {
		if ef.Name == name {
			cs.editEntryIndex = i
			return i
		}
	}
	t.Fatalf("entry field %q not found", name)
	return -1
}

func makeConfigScreenDirty(t *testing.T, m Model) Model {
	t.Helper()

	cs := m.configScreen
	setConfigFieldSelection(t, cs, "compression", "compression.enabled")
	m = sendConfigKey(m, " ")
	if !m.configScreen.dirty {
		t.Fatal("dirty should be true after toggle")
	}
	return m
}
