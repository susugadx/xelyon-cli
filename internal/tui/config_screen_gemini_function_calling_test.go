package tui

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui/configscreen"
)

func newGeminiConfigScreenTestModel(cfg *config.Config) Model {
	agent := &stubAgent{providerName: "gemini", providerConfigKey: "gemini"}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	m.configScreen = configscreen.New(cfg)
	return m
}

func TestConfigScreen_GeminiDefaultModelRejectsUnsupportedFunctionCallingModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "gemini"
	m := newGeminiConfigScreenTestModel(cfg)
	cs := configTestScreen(t, m)

	selectConfigField(t, &m, "provider", "default_model")
	previousConfig := cs.ConfigSnapshot()
	previousDefault := previousConfig.DefaultModel
	previousGeminiDefault := previousConfig.GetExplicitProviderDefaultModel("gemini")

	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
	if got := cs.Snapshot().EditMode; got != editInput {
		t.Fatalf("editMode = %d, want editInput", got)
	}
	setConfigInputValue(t, &m, "gemini-2.0-flash-lite")

	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

	currentConfig := cs.ConfigSnapshot()
	if currentConfig.DefaultModel != previousDefault {
		t.Fatalf("DefaultModel = %q, want unchanged %q", currentConfig.DefaultModel, previousDefault)
	}
	if got := currentConfig.GetExplicitProviderDefaultModel("gemini"); got != previousGeminiDefault {
		t.Fatalf("provider_models.gemini.default_model = %q, want unchanged %q", got, previousGeminiDefault)
	}
	snapshot := cs.Snapshot()
	if snapshot.EditMode != editInput {
		t.Fatalf("editMode = %d, want editInput after rejected value", snapshot.EditMode)
	}
	if snapshot.SaveStatus != statusFailed || !strings.Contains(snapshot.SaveError, "provider_models.gemini.default_model") {
		t.Fatalf("saveStatus/saveError = %d/%q, want Gemini default_model validation error", snapshot.SaveStatus, snapshot.SaveError)
	}
}

func TestConfigScreen_GeminiProviderCatalogRejectsUnsupportedFunctionCallingModel(t *testing.T) {
	m := enterStructMapEntryForKey(t, "provider_models", "gemini")
	cs := configTestScreen(t, m)
	selectConfigEntryField(t, &m, "catalog_model")

	previous := cs.ConfigSnapshot().ProviderModels["gemini"].CatalogModel
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)
	if got := cs.Snapshot().EditEntryFieldEdit; got != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want input", got)
	}
	setConfigInputValue(t, &m, "models/gemini-2.0-flash-lite")

	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

	if got := cs.ConfigSnapshot().ProviderModels["gemini"].CatalogModel; got != previous {
		t.Fatalf("provider_models.gemini.catalog_model = %q, want unchanged %q", got, previous)
	}
	snapshot := cs.Snapshot()
	if snapshot.EditEntryFieldEdit != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want input after rejected value", snapshot.EditEntryFieldEdit)
	}
	if snapshot.SaveStatus != statusFailed || !strings.Contains(snapshot.SaveError, "provider_models.gemini.catalog_model") {
		t.Fatalf("saveStatus/saveError = %d/%q, want Gemini catalog_model validation error", snapshot.SaveStatus, snapshot.SaveError)
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
	m.configScreen = configscreen.New(cfg)
	setConfigDirtyForTest(t, &m, true)

	m = saveConfigAndWait(t, m)
	cs := m.configScreen

	agent.mu.RLock()
	saved := agent.lastSavedConfig
	agent.mu.RUnlock()
	if saved != nil {
		t.Fatalf("lastSavedConfig = %#v, want nil when Gemini validation fails", saved)
	}
	snapshot := cs.Snapshot()
	if snapshot.SaveStatus != statusFailed || !strings.Contains(snapshot.SaveError, "provider_models.gemini.default_model") {
		t.Fatalf("saveStatus/saveError = %d/%q, want Gemini validation failure", snapshot.SaveStatus, snapshot.SaveError)
	}
}
