package tui

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestConfigScreen_ResetToDefault(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	for i, cat := range cs.categories {
		if cat.Name == "compression" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == "compression.trigger_percent" {
			cs.fieldIndex = i
			break
		}
	}

	if err := config.SetFieldValue(cs.cfg, "compression.trigger_percent", 50); err != nil {
		t.Fatalf("SetFieldValue failed: %v", err)
	}
	cs.dirty = true
	cs.refreshCategories()

	m = sendConfigKey(m, "r")
	cs = m.configScreen

	val, _ := config.GetFieldValue(cs.cfg, "compression.trigger_percent")
	defCfg := config.DefaultConfig()
	defVal, _ := config.GetFieldValue(defCfg, "compression.trigger_percent")
	if val != defVal {
		t.Fatalf("value after reset = %v, want default %v", val, defVal)
	}
}

func TestConfigScreen_FilterFields(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	for i, cat := range cs.categories {
		if cat.Name == "execution" {
			cs.catIndex = i
			break
		}
	}

	m = sendConfigKey(m, "/")
	cs = m.configScreen
	if !cs.filterMode {
		t.Fatal("filterMode should be true")
	}

	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.filterMode {
		t.Fatal("filterMode should be false after esc")
	}
}
