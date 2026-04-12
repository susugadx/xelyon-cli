package tui

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDefaultModelSyncProvider_UsesCurrentSessionOwnerWhenDefaultProviderOnlyChangesAliasSpelling(t *testing.T) {
	agent := &stubAgent{providerName: "deepseek"}
	m := newModelWithViewport(agent)

	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "anthropic"

	cs := newConfigScreen(cfg)
	cs.initialDefaultProvider = "claude"
	m.configScreen = cs

	if got := m.defaultModelSyncProvider(); got != "deepseek" {
		t.Fatalf("defaultModelSyncProvider() = %q, want %q", got, "deepseek")
	}
}

func TestDefaultModelSyncProvider_UsesNewDefaultProviderWhenRuntimeChanges(t *testing.T) {
	agent := &stubAgent{providerName: "deepseek"}
	m := newModelWithViewport(agent)

	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "openai"

	cs := newConfigScreen(cfg)
	cs.initialDefaultProvider = "deepseek"
	m.configScreen = cs

	if got := m.defaultModelSyncProvider(); got != "openai" {
		t.Fatalf("defaultModelSyncProvider() = %q, want %q", got, "openai")
	}
}

func TestSyncEditedProviderDefaultModel_UsesCurrentSessionOwnerWhenDefaultProviderOnlyChangesAliasSpelling(t *testing.T) {
	agent := &stubAgent{providerName: "deepseek"}
	m := newModelWithViewport(agent)

	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.DefaultModel = "claude-custom"
	cfg.SetProviderModelConfig("deepseek", config.ProviderModelConfig{DefaultModel: "deepseek-chat"})

	cs := newConfigScreen(cfg)
	cs.initialDefaultProvider = "claude"
	m.configScreen = cs

	m.syncEditedProviderDefaultModel()

	saved := cs.cfg.ProviderModelsForSave()
	if pm, ok := saved["deepseek"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want deepseek override", saved)
	} else if pm.DefaultModel != "claude-custom" {
		t.Fatalf("ProviderModelsForSave()[deepseek].DefaultModel = %q, want %q", pm.DefaultModel, "claude-custom")
	}
	if _, ok := saved["anthropic"]; ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want anthropic alias to remain untouched", saved)
	}
}

func TestDefaultModelSyncProvider_UsesSessionProviderConfigKeyWhenDefaultProviderDiffers(t *testing.T) {
	agent := &stubAgent{providerName: "claude", providerConfigKey: "anthropic"}
	m := newModelWithViewport(agent)

	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "deepseek"

	cs := newConfigScreen(cfg)
	cs.initialDefaultProvider = "deepseek"
	m.configScreen = cs

	if got := m.defaultModelSyncProvider(); got != "anthropic" {
		t.Fatalf("defaultModelSyncProvider() = %q, want %q", got, "anthropic")
	}
}

func TestDefaultModelSyncProvider_UsesSessionProviderConfigKeyWhenDefaultProviderSpellingIsUnchanged(t *testing.T) {
	agent := &stubAgent{providerName: "claude", providerConfigKey: "anthropic"}
	m := newModelWithViewport(agent)

	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "claude"

	cs := newConfigScreen(cfg)
	cs.initialDefaultProvider = "claude"
	m.configScreen = cs

	if got := m.defaultModelSyncProvider(); got != "anthropic" {
		t.Fatalf("defaultModelSyncProvider() = %q, want %q", got, "anthropic")
	}
}

func TestSyncEditedProviderDefaultModel_PreservesSessionAnthropicAliasWhenDefaultProviderDiffers(t *testing.T) {
	agent := &stubAgent{providerName: "claude", providerConfigKey: "anthropic"}
	m := newModelWithViewport(agent)

	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.DefaultModel = "anthropic-new"
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-old"},
		"claude":    {DefaultModel: "claude-old"},
	})

	cs := newConfigScreen(cfg)
	cs.initialDefaultProvider = "deepseek"
	m.configScreen = cs

	m.syncEditedProviderDefaultModel()

	saved := cs.cfg.ProviderModelsForSave()
	if pm, ok := saved["anthropic"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want anthropic override", saved)
	} else if pm.DefaultModel != "anthropic-new" {
		t.Fatalf("ProviderModelsForSave()[anthropic].DefaultModel = %q, want %q", pm.DefaultModel, "anthropic-new")
	}
	if pm, ok := saved["claude"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want claude sibling preserved", saved)
	} else if pm.DefaultModel != "claude-old" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want %q", pm.DefaultModel, "claude-old")
	}
}

func TestSyncEditedProviderDefaultModel_CreatesAnthropicEntryWhenDefaultProviderSpellingIsUnchanged(t *testing.T) {
	agent := &stubAgent{providerName: "claude", providerConfigKey: "anthropic"}
	m := newModelWithViewport(agent)

	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "claude"
	cfg.DefaultModel = "anthropic-new"
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"claude": {DefaultModel: "claude-old"},
	})

	cs := newConfigScreen(cfg)
	cs.initialDefaultProvider = "claude"
	m.configScreen = cs

	m.syncEditedProviderDefaultModel()

	saved := cs.cfg.ProviderModelsForSave()
	if pm, ok := saved["anthropic"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want anthropic override created", saved)
	} else if pm.DefaultModel != "anthropic-new" {
		t.Fatalf("ProviderModelsForSave()[anthropic].DefaultModel = %q, want %q", pm.DefaultModel, "anthropic-new")
	}
	if pm, ok := saved["claude"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want claude sibling preserved", saved)
	} else if pm.DefaultModel != "claude-old" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want %q", pm.DefaultModel, "claude-old")
	}
}
