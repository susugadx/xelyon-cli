package tui

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestConfigScreen_ResetToDefault(t *testing.T) {
	cfg := config.DefaultConfig()
	if err := config.SetFieldValue(cfg, "compression.trigger_percent", 50); err != nil {
		t.Fatalf("SetFieldValue failed: %v", err)
	}
	m := newConfigTestModelWithConfig(cfg)
	selectConfigField(t, &m, "compression", "compression.trigger_percent")

	m = sendConfigKey(m, "r")
	cs := configTestScreen(t, m)

	val, _ := config.GetFieldValue(cs.ConfigSnapshot(), "compression.trigger_percent")
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
	if !cs.Snapshot().FilterMode {
		t.Fatal("filterMode should be true")
	}

	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.Snapshot().FilterMode {
		t.Fatal("filterMode should be false after esc")
	}
}
