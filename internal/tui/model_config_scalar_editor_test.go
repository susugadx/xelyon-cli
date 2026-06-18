package tui

import (
	"reflect"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestConfigScreen_BoolToggle(t *testing.T) {
	m := newConfigTestModel()
	selectConfigField(t, &m, "compression", "compression.enabled")

	field := selectedConfigField(t, m)
	current, _ := field.Current.(bool)

	m = sendConfigKey(m, " ")

	cs := configTestScreen(t, m)
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
	openConfigSelectEditor(t, &m, "execution", "execution.mode")

	m = sendConfigKeys(m, "j", "enter")
	cs := configTestScreen(t, m)
	if cs.editMode != editNone {
		t.Fatalf("editMode after select = %d, want editNone", cs.editMode)
	}
	if !cs.dirty {
		t.Fatal("dirty should be true after select edit")
	}
}

func TestConfigScreen_StringEdit(t *testing.T) {
	m := newConfigTestModel()
	openConfigInputEditor(t, &m, "provider", "default_model")

	m = sendConfigKey(m, "esc")
	cs := configTestScreen(t, m)
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
			selectConfigField(t, &m, tt.category, tt.path)

			cs := configTestScreen(t, m)
			before, err := config.GetFieldValue(cs.cfg, tt.path)
			if err != nil {
				t.Fatalf("GetFieldValue before: %v", err)
			}

			openConfigInputEditor(t, &m, tt.category, tt.path)

			setConfigInputValue(t, &m, "")
			m = sendConfigKey(m, "enter")
			cs = configTestScreen(t, m)

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
	selectConfigField(t, &m, "compression", "compression.enabled")
	cs := configTestScreen(t, m)
	before, _ := config.GetFieldValue(cs.cfg, "compression.enabled")
	m = sendConfigKey(m, " ")
	after, _ := config.GetFieldValue(m.configScreen.cfg, "compression.enabled")
	if before == after {
		t.Fatal("Space should toggle bool")
	}

	selectConfigField(t, &m, "execution", "execution.mode")
	m = sendConfigKey(m, " ")
	cs = configTestScreen(t, m)
	if cs.editMode != editNone {
		t.Fatalf("Space on select should be no-op, but editMode = %d", cs.editMode)
	}

	selectConfigField(t, &m, "provider", "default_model")
	m = sendConfigKey(m, " ")
	cs = configTestScreen(t, m)
	if cs.editMode != editNone {
		t.Fatalf("Space on string should be no-op, but editMode = %d", cs.editMode)
	}

	selectConfigField(t, &m, "lsp", "lsp.servers")
	m = sendConfigKey(m, " ")
	cs = configTestScreen(t, m)
	if cs.editMode != editNone {
		t.Fatalf("Space on structmap should be no-op, but editMode = %d", cs.editMode)
	}

	selectConfigField(t, &m, "execution", "execution.mode")
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
	if cs.editMode != editSelect {
		t.Fatalf("Enter on select should start edit, but editMode = %d", cs.editMode)
	}
}
