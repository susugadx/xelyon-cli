package tui

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func newGeminiConfigScreenTestModel(cfg *config.Config) Model {
	agent := &stubAgent{providerName: "gemini", providerConfigKey: "gemini"}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)
	return m
}

func TestConfigScreen_GeminiDefaultModelRejectsUnsupportedFunctionCallingModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "gemini"
	m := newGeminiConfigScreenTestModel(cfg)
	cs := configTestScreen(t, m)

	selectConfigField(t, &m, "provider", "default_model")
	previousDefault := cs.cfg.DefaultModel
	previousGeminiDefault := cs.cfg.GetExplicitProviderDefaultModel("gemini")

	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
	if cs.editMode != editInput {
		t.Fatalf("editMode = %d, want editInput", cs.editMode)
	}
	setConfigInputValue(t, &m, "gemini-2.0-flash-lite")

	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

	if cs.cfg.DefaultModel != previousDefault {
		t.Fatalf("DefaultModel = %q, want unchanged %q", cs.cfg.DefaultModel, previousDefault)
	}
	if got := cs.cfg.GetExplicitProviderDefaultModel("gemini"); got != previousGeminiDefault {
		t.Fatalf("provider_models.gemini.default_model = %q, want unchanged %q", got, previousGeminiDefault)
	}
	if cs.editMode != editInput {
		t.Fatalf("editMode = %d, want editInput after rejected value", cs.editMode)
	}
	if cs.saveStatus != statusFailed || !strings.Contains(cs.saveError, "provider_models.gemini.default_model") {
		t.Fatalf("saveStatus/saveError = %d/%q, want Gemini default_model validation error", cs.saveStatus, cs.saveError)
	}
}

func TestConfigScreen_GeminiProviderCatalogRejectsUnsupportedFunctionCallingModel(t *testing.T) {
	m := enterStructMapEntryForKey(t, "provider_models", "gemini")
	cs := configTestScreen(t, m)
	selectConfigEntryField(t, &m, "catalog_model")

	previous := cs.cfg.ProviderModels["gemini"].CatalogModel
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
	if cs.editEntryFieldEdit != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want input", cs.editEntryFieldEdit)
	}
	setConfigInputValue(t, &m, "models/gemini-2.0-flash-lite")

	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

	if got := cs.cfg.ProviderModels["gemini"].CatalogModel; got != previous {
		t.Fatalf("provider_models.gemini.catalog_model = %q, want unchanged %q", got, previous)
	}
	if cs.editEntryFieldEdit != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want input after rejected value", cs.editEntryFieldEdit)
	}
	if cs.saveStatus != statusFailed || !strings.Contains(cs.saveError, "provider_models.gemini.catalog_model") {
		t.Fatalf("saveStatus/saveError = %d/%q, want Gemini catalog_model validation error", cs.saveStatus, cs.saveError)
	}
}

func TestConfigScreen_SaveRejectsUnsupportedGeminiConfigBeforeSync(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("gemini", config.ProviderModelConfig{
		DefaultModel: "gemini-2.0-flash-lite",
	})
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)
	setConfigDirtyForTest(t, &m, true)

	m = saveConfigAndWait(t, m)
	cs := m.configScreen

	agent.mu.RLock()
	saved := agent.lastSavedConfig
	agent.mu.RUnlock()
	if saved != nil {
		t.Fatalf("lastSavedConfig = %#v, want nil when Gemini validation fails", saved)
	}
	if cs.saveStatus != statusFailed || !strings.Contains(cs.saveError, "provider_models.gemini.default_model") {
		t.Fatalf("saveStatus/saveError = %d/%q, want Gemini validation failure", cs.saveStatus, cs.saveError)
	}
}
