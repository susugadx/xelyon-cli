package tui

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestConfigScreen_ProviderModelDirectEditStillDoesNotGetOverwritten(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "global-model"
	if pm, ok := cfg.ProviderModels["openai"]; ok {
		pm.DefaultModel = "openai-specific"
		cfg.ProviderModels["openai"] = pm
	}
	m := newConfigTestModelWithConfig(cfg)
	setConfigDirtyForTest(t, &m, true)

	m = saveConfigAndWait(t, m)

	pm := m.configScreen.ConfigSnapshot().ProviderModels["openai"]
	if pm.DefaultModel != "openai-specific" {
		t.Fatalf("ProviderModels[openai].DefaultModel = %q, want \"openai-specific\"", pm.DefaultModel)
	}
}
