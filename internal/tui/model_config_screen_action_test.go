package tui

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestConfigScreen_ResetToDefault(t *testing.T) {
	m := newConfigTestModel()
	selectConfigField(t, &m, "compression", "compression.trigger_percent")

	cs := configTestScreen(t, m)
	if err := config.SetFieldValue(cs.cfg, "compression.trigger_percent", 50); err != nil {
		t.Fatalf("SetFieldValue failed: %v", err)
	}
	setConfigDirtyForTest(t, &m, true)
	cs.refreshCategories()

	m = sendConfigKey(m, "r")
	cs = configTestScreen(t, m)

	val, _ := config.GetFieldValue(cs.cfg, "compression.trigger_percent")
	defCfg := config.DefaultConfig()
	defVal, _ := config.GetFieldValue(defCfg, "compression.trigger_percent")
	if val != defVal {
		t.Fatalf("value after reset = %v, want default %v", val, defVal)
	}
}

func TestConfigScreen_FilterFields(t *testing.T) {
	m := newConfigTestModel()
	selectConfigCategory(t, &m, "execution")

	m = sendConfigKey(m, "/")
	cs := configTestScreen(t, m)
	if !cs.filterMode {
		t.Fatal("filterMode should be true")
	}

	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.filterMode {
		t.Fatal("filterMode should be false after esc")
	}
}
