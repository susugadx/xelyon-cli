package tui

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func selectConfigCategory(t *testing.T, m *Model, categoryName string) {
	t.Helper()
	cs := configTestScreen(t, *m)
	for i, cat := range cs.categories {
		if cat.Name == categoryName {
			cs.catIndex = i
			cs.fieldIndex = 0
			cs.fieldScroll = 0
			return
		}
	}
	t.Fatalf("category %q not found", categoryName)
}

func selectConfigField(t *testing.T, m *Model, categoryName, fieldPath string) {
	t.Helper()
	selectConfigCategory(t, m, categoryName)

	cs := configTestScreen(t, *m)
	cs.activePane = paneField
	for i, f := range cs.filteredFields() {
		if f.Path == fieldPath {
			cs.fieldIndex = i
			cs.fieldScroll = 0
			return
		}
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
	field := configTestScreen(t, m).selectedField()
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
	configTestScreen(t, *m).editInput.SetValue(value)
}

func setConfigSliceInputValue(t *testing.T, m *Model, value string) {
	t.Helper()
	configTestScreen(t, *m).editSliceInput.SetValue(value)
}

func setConfigStructInputValue(t *testing.T, m *Model, value string) {
	t.Helper()
	configTestScreen(t, *m).editStructInput.SetValue(value)
}

func selectConfigOption(t *testing.T, m Model, categoryName, fieldPath, option string) Model {
	t.Helper()

	openConfigSelectEditor(t, &m, categoryName, fieldPath)
	field := selectedConfigField(t, m)

	cs := configTestScreen(t, m)
	for i, candidate := range field.Options {
		if candidate == option {
			cs.editSelect = i
			return sendConfigKey(m, "enter")
		}
	}
	t.Fatalf("option %q not found in %s", option, fieldPath)
	return m
}
