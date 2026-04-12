package tui

import "testing"

func TestConfigScreen_ProviderModelDirectEditStillDoesNotGetOverwritten(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	cs.cfg.DefaultProvider = "openai"
	cs.cfg.DefaultModel = "global-model"
	if pm, ok := cs.cfg.ProviderModels["openai"]; ok {
		pm.DefaultModel = "openai-specific"
		cs.cfg.ProviderModels["openai"] = pm
	}
	cs.dirty = true

	m = saveConfigAndWait(t, m)
	cs = m.configScreen

	pm := cs.cfg.ProviderModels["openai"]
	if pm.DefaultModel != "openai-specific" {
		t.Fatalf("ProviderModels[openai].DefaultModel = %q, want \"openai-specific\"", pm.DefaultModel)
	}
}
