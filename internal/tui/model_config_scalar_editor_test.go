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
	newVal, _ := config.GetFieldValue(cs.ConfigSnapshot(), "compression.enabled")
	if newVal.(bool) == current {
		t.Fatalf("bool value did not toggle: still %v", current)
	}
	snapshot := cs.Snapshot()
	if !snapshot.Dirty {
		t.Fatal("dirty should be true after toggle")
	}
	if snapshot.SaveStatus != statusModified {
		t.Fatalf("saveStatus = %d, want statusModified(%d)", snapshot.SaveStatus, statusModified)
	}
}

func TestConfigScreen_SelectEdit(t *testing.T) {
	m := newConfigTestModel()
	openConfigSelectEditor(t, &m, "execution", "execution.mode")

	m = sendConfigKeys(m, "j", "enter")
	cs := configTestScreen(t, m)
	snapshot := cs.Snapshot()
	if snapshot.EditMode != editNone {
		t.Fatalf("editMode after select = %d, want editNone", snapshot.EditMode)
	}
	if !snapshot.Dirty {
		t.Fatal("dirty should be true after select edit")
	}
}

func TestConfigScreen_StringEdit(t *testing.T) {
	m := newConfigTestModel()
	openConfigInputEditor(t, &m, "provider", "default_model")

	m = sendConfigKey(m, "esc")
	cs := configTestScreen(t, m)
	if got := cs.Snapshot().EditMode; got != editNone {
		t.Fatalf("editMode after esc = %d, want editNone", got)
	}
}

func TestConfigScreen_RawEnterAppliesStringEdit(t *testing.T) {
	for _, tt := range enterFallbackKeyCases() {
		t.Run(tt.name, func(t *testing.T) {
			m := newConfigTestModel()
			openConfigInputEditor(t, &m, "provider", "default_model")
			setConfigInputValue(t, &m, "raw-enter-model")

			m = sendConfigKeyMsg(m, tt.key)

			cs := configTestScreen(t, m)
			snapshot := cs.Snapshot()
			if snapshot.EditMode != editNone {
				t.Fatalf("editMode after raw Enter = %d, want editNone", snapshot.EditMode)
			}
			if !snapshot.Dirty {
				t.Fatal("dirty should be true after raw Enter applies string edit")
			}
			got, err := config.GetFieldValue(cs.ConfigSnapshot(), "default_model")
			if err != nil {
				t.Fatalf("GetFieldValue: %v", err)
			}
			if got != "raw-enter-model" {
				t.Fatalf("default_model = %q, want raw-enter-model", got)
			}
		})
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
			before, err := config.GetFieldValue(cs.ConfigSnapshot(), tt.path)
			if err != nil {
				t.Fatalf("GetFieldValue before: %v", err)
			}

			openConfigInputEditor(t, &m, tt.category, tt.path)

			setConfigInputValue(t, &m, "")
			m = sendConfigKey(m, "enter")
			cs = configTestScreen(t, m)

			snapshot := cs.Snapshot()
			if snapshot.EditMode != editInput {
				t.Fatalf("editMode after empty input = %d, want editInput(%d)", snapshot.EditMode, editInput)
			}
			if snapshot.Dirty {
				t.Fatal("dirty should remain false after empty numeric input")
			}
			if snapshot.SaveStatus != statusSaved {
				t.Fatalf("saveStatus = %d, want statusSaved(%d)", snapshot.SaveStatus, statusSaved)
			}

			after, err := config.GetFieldValue(cs.ConfigSnapshot(), tt.path)
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
	before, _ := config.GetFieldValue(cs.ConfigSnapshot(), "compression.enabled")
	m = sendConfigKey(m, " ")
	after, _ := config.GetFieldValue(m.configScreen.ConfigSnapshot(), "compression.enabled")
	if before == after {
		t.Fatal("Space should toggle bool")
	}

	selectConfigField(t, &m, "execution", "execution.mode")
	m = sendConfigKey(m, " ")
	cs = configTestScreen(t, m)
	if got := cs.Snapshot().EditMode; got != editNone {
		t.Fatalf("Space on select should be no-op, but editMode = %d", got)
	}

	selectConfigField(t, &m, "provider", "default_model")
	m = sendConfigKey(m, " ")
	cs = configTestScreen(t, m)
	if got := cs.Snapshot().EditMode; got != editNone {
		t.Fatalf("Space on string should be no-op, but editMode = %d", got)
	}

	selectConfigField(t, &m, "lsp", "lsp.servers")
	m = sendConfigKey(m, " ")
	cs = configTestScreen(t, m)
	if got := cs.Snapshot().EditMode; got != editNone {
		t.Fatalf("Space on structmap should be no-op, but editMode = %d", got)
	}

	selectConfigField(t, &m, "execution", "execution.mode")
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
	if got := cs.Snapshot().EditMode; got != editSelect {
		t.Fatalf("Enter on select should start edit, but editMode = %d", got)
	}
}
