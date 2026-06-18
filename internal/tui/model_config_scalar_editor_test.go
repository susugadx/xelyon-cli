package tui

import (
	"reflect"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestConfigScreen_BoolToggle(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen
	for i, cat := range cs.categories {
		if cat.Name == "compression" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	cs.fieldIndex = 0

	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == "compression.enabled" {
			cs.fieldIndex = i
			break
		}
	}

	field := cs.selectedField()
	if field == nil {
		t.Fatal("selectedField is nil")
	}
	current, _ := field.Current.(bool)

	m = sendConfigKey(m, " ")

	cs = m.configScreen
	newVal, _ := config.GetFieldValue(cs.cfg, "compression.enabled")
	if newVal.(bool) == current {
		t.Fatalf("bool value did not toggle: still %v", current)
	}
	if !cs.dirty {
		t.Fatal("dirty should be true after toggle")
	}
	if cs.saveStatus != statusModified {
		t.Fatalf("saveStatus = %d, want statusModified(%d)", cs.saveStatus, statusModified)
	}
}

func TestConfigScreen_SelectEdit(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	for i, cat := range cs.categories {
		if cat.Name == "execution" {
			cs.catIndex = i
			break
		}
	}

	cs.activePane = paneField
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == "execution.mode" {
			cs.fieldIndex = i
			break
		}
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editSelect {
		t.Fatalf("editMode = %d, want editSelect(%d)", cs.editMode, editSelect)
	}

	m = sendConfigKeys(m, "j", "enter")
	cs = m.configScreen
	if cs.editMode != editNone {
		t.Fatalf("editMode after select = %d, want editNone", cs.editMode)
	}
	if !cs.dirty {
		t.Fatal("dirty should be true after select edit")
	}
}

func TestConfigScreen_StringEdit(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	for i, cat := range cs.categories {
		if cat.Name == "provider" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == "default_model" {
			cs.fieldIndex = i
			break
		}
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editInput {
		t.Fatalf("editMode = %d, want editInput(%d)", cs.editMode, editInput)
	}

	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.editMode != editNone {
		t.Fatalf("editMode after esc = %d, want editNone", cs.editMode)
	}
}

func TestConfigScreen_NumericEmptyInputDoesNotApply(t *testing.T) {
	tests := []struct {
		name     string
		category string
		path     string
	}{
		{
			name:     "int",
			category: "compression",
			path:     "compression.trigger_percent",
		},
		{
			name:     "float",
			category: "project_map",
			path:     "project_map.context_ratio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newConfigTestModel()
			cs := m.configScreen
			setConfigFieldSelection(t, cs, tt.category, tt.path)

			before, err := config.GetFieldValue(cs.cfg, tt.path)
			if err != nil {
				t.Fatalf("GetFieldValue before: %v", err)
			}

			m = sendConfigKey(m, "enter")
			cs = m.configScreen
			if cs.editMode != editInput {
				t.Fatalf("editMode = %d, want editInput(%d)", cs.editMode, editInput)
			}

			cs.editInput.SetValue("")
			m = sendConfigKey(m, "enter")
			cs = m.configScreen

			if cs.editMode != editInput {
				t.Fatalf("editMode after empty input = %d, want editInput(%d)", cs.editMode, editInput)
			}
			if cs.dirty {
				t.Fatal("dirty should remain false after empty numeric input")
			}
			if cs.saveStatus != statusSaved {
				t.Fatalf("saveStatus = %d, want statusSaved(%d)", cs.saveStatus, statusSaved)
			}

			after, err := config.GetFieldValue(cs.cfg, tt.path)
			if err != nil {
				t.Fatalf("GetFieldValue after: %v", err)
			}
			if !reflect.DeepEqual(after, before) {
				t.Fatalf("value changed: before=%v after=%v", before, after)
			}
		})
	}
}

func TestConfigScreen_SpaceBoolOnly(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	for i, cat := range cs.categories {
		if cat.Name == "compression" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	for i, f := range cs.filteredFields() {
		if f.Path == "compression.enabled" {
			cs.fieldIndex = i
			break
		}
	}
	before, _ := config.GetFieldValue(cs.cfg, "compression.enabled")
	m = sendConfigKey(m, " ")
	after, _ := config.GetFieldValue(m.configScreen.cfg, "compression.enabled")
	if before == after {
		t.Fatal("Space should toggle bool")
	}

	cs = m.configScreen
	for i, cat := range cs.categories {
		if cat.Name == "execution" {
			cs.catIndex = i
			break
		}
	}
	cs.fieldIndex = 0
	cs.fieldScroll = 0
	for i, f := range cs.filteredFields() {
		if f.Path == "execution.mode" {
			cs.fieldIndex = i
			break
		}
	}
	m = sendConfigKey(m, " ")
	cs = m.configScreen
	if cs.editMode != editNone {
		t.Fatalf("Space on select should be no-op, but editMode = %d", cs.editMode)
	}

	for i, cat := range cs.categories {
		if cat.Name == "provider" {
			cs.catIndex = i
			break
		}
	}
	cs.fieldIndex = 0
	cs.fieldScroll = 0
	for i, f := range cs.filteredFields() {
		if f.Path == "default_model" {
			cs.fieldIndex = i
			break
		}
	}
	m = sendConfigKey(m, " ")
	cs = m.configScreen
	if cs.editMode != editNone {
		t.Fatalf("Space on string should be no-op, but editMode = %d", cs.editMode)
	}

	for i, cat := range cs.categories {
		if cat.Name == "lsp" {
			cs.catIndex = i
			break
		}
	}
	cs.fieldIndex = 0
	cs.fieldScroll = 0
	for i, f := range cs.filteredFields() {
		if f.Path == "lsp.servers" {
			cs.fieldIndex = i
			break
		}
	}
	m = sendConfigKey(m, " ")
	cs = m.configScreen
	if cs.editMode != editNone {
		t.Fatalf("Space on structmap should be no-op, but editMode = %d", cs.editMode)
	}

	for i, cat := range cs.categories {
		if cat.Name == "execution" {
			cs.catIndex = i
			break
		}
	}
	cs.fieldIndex = 0
	cs.fieldScroll = 0
	for i, f := range cs.filteredFields() {
		if f.Path == "execution.mode" {
			cs.fieldIndex = i
			break
		}
	}
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editSelect {
		t.Fatalf("Enter on select should start edit, but editMode = %d", cs.editMode)
	}
}
