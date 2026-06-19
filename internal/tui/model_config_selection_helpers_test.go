package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func selectConfigCategory(t *testing.T, m *Model, categoryName string) {
	t.Helper()
	for configSnapshot(t, *m).activePane != paneCategory {
		*m = sendConfigKey(*m, "left")
	}
	snapshot := configTestScreen(t, *m).Snapshot()
	target := -1
	for i, name := range snapshot.CategoryNames {
		if name == categoryName {
			target = i
			break
		}
	}
	if target < 0 {
		t.Fatalf("category %q not found", categoryName)
	}
	for configSnapshot(t, *m).categoryIndex < target {
		*m = sendConfigKey(*m, "down")
	}
	for configSnapshot(t, *m).categoryIndex > target {
		*m = sendConfigKey(*m, "up")
	}
}

func selectConfigField(t *testing.T, m *Model, categoryName, fieldPath string) {
	t.Helper()
	selectConfigCategory(t, m, categoryName)

	if configSnapshot(t, *m).activePane == paneCategory {
		*m = sendConfigKey(*m, "right")
	}
	fields := configTestScreen(t, *m).Snapshot().FilteredFields
	target := -1
	for i, f := range fields {
		if f.Path == fieldPath {
			target = i
			break
		}
	}
	if target >= 0 {
		for configSnapshot(t, *m).fieldIndex < target {
			*m = sendConfigKey(*m, "down")
		}
		for configSnapshot(t, *m).fieldIndex > target {
			*m = sendConfigKey(*m, "up")
		}
		return
	}
	t.Fatalf("field %q not found in category %q", fieldPath, categoryName)
}

func selectConfigFieldByPath(t *testing.T, m *Model, fieldPath string) {
	t.Helper()
	selectConfigField(t, m, configCategoryForFieldPath(fieldPath), fieldPath)
}

func configCategoryForFieldPath(fieldPath string) string {
	if fieldPath == "provider_models" {
		return "provider"
	}
	if before, _, ok := strings.Cut(fieldPath, "."); ok {
		return before
	}
	return fieldPath
}

func selectedConfigField(t *testing.T, m Model) config.ConfigField {
	t.Helper()
	field := configTestScreen(t, m).Snapshot().SelectedField
	if field == nil {
		t.Fatal("selectedField is nil")
	}
	return *field
}

func openConfigEditor(t *testing.T, m *Model, categoryName, fieldPath string, want configEditMode) {
	t.Helper()
	selectConfigField(t, m, categoryName, fieldPath)
	*m = sendConfigKey(*m, "enter")
	if got := configSnapshot(t, *m).editMode; got != want {
		t.Fatalf("editMode = %d, want %d", got, want)
	}
}

func openConfigInputEditor(t *testing.T, m *Model, categoryName, fieldPath string) {
	t.Helper()
	openConfigEditor(t, m, categoryName, fieldPath, editInput)
}

func openConfigSelectEditor(t *testing.T, m *Model, categoryName, fieldPath string) {
	t.Helper()
	openConfigEditor(t, m, categoryName, fieldPath, editSelect)
}

func openConfigSliceEditor(t *testing.T, m *Model, categoryName, fieldPath string) {
	t.Helper()
	openConfigEditor(t, m, categoryName, fieldPath, editSlice)
}

func setConfigInputValue(t *testing.T, m *Model, value string) {
	t.Helper()
	*m = sendConfigKeyMsg(*m, tea.KeyMsg{Type: tea.KeyCtrlU})
	if value != "" {
		*m = sendConfigKeyMsg(*m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)})
	}
}

func setConfigSliceInputValue(t *testing.T, m *Model, value string) {
	t.Helper()
	*m = sendConfigKeyMsg(*m, tea.KeyMsg{Type: tea.KeyCtrlU})
	if value != "" {
		*m = sendConfigKeyMsg(*m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)})
	}
}

func setConfigStructInputValue(t *testing.T, m *Model, value string) {
	t.Helper()
	*m = sendConfigKeyMsg(*m, tea.KeyMsg{Type: tea.KeyCtrlU})
	if value != "" {
		*m = sendConfigKeyMsg(*m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)})
	}
}

func selectConfigOption(t *testing.T, m Model, categoryName, fieldPath, option string) Model {
	t.Helper()

	openConfigSelectEditor(t, &m, categoryName, fieldPath)
	field := selectedConfigField(t, m)

	snapshot := configTestScreen(t, m).Snapshot()
	for i, candidate := range field.Options {
		if candidate == option {
			for snapshot.EditSelect < i {
				m = sendConfigKey(m, "down")
				snapshot = configTestScreen(t, m).Snapshot()
			}
			for snapshot.EditSelect > i {
				m = sendConfigKey(m, "up")
				snapshot = configTestScreen(t, m).Snapshot()
			}
			return sendConfigKey(m, "enter")
		}
	}
	t.Fatalf("option %q not found in %s", option, fieldPath)
	return m
}
