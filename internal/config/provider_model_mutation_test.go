package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetProviderModelConfig_AnthropicReusesExistingClaudeKey(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"claude": {
			DefaultModel:     "claude-custom",
			AnthropicVersion: "2099-01-01",
		},
	})

	cfg.SetProviderModelConfig("anthropic", ProviderModelConfig{
		DefaultModel:     "anthropic-updated",
		MaxOutputTokens:  4321,
		AnthropicVersion: "2100-01-01",
	})

	saved := cfg.ProviderModelsForSave()
	if len(saved) != 1 {
		t.Fatalf("len(ProviderModelsForSave()) = %d, want 1", len(saved))
	}
	if _, ok := saved["anthropic"]; ok {
		t.Fatal("ProviderModelsForSave() should not keep duplicate anthropic entry when claude already existed")
	}

	pm, ok := saved["claude"]
	if !ok {
		t.Fatal("ProviderModelsForSave() should reuse existing claude key")
	}
	if pm.DefaultModel != "anthropic-updated" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want %q", pm.DefaultModel, "anthropic-updated")
	}
	if pm.MaxOutputTokens != 4321 {
		t.Fatalf("ProviderModelsForSave()[claude].MaxOutputTokens = %d, want %d", pm.MaxOutputTokens, 4321)
	}
	if pm.AnthropicVersion != "2100-01-01" {
		t.Fatalf("ProviderModelsForSave()[claude].AnthropicVersion = %q, want %q", pm.AnthropicVersion, "2100-01-01")
	}
}

func TestSetProviderModelConfig_PreservesClaudeSiblingWhenUpdatingAnthropicEntry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"claude": {
			DefaultModel:     "claude-custom",
			AnthropicVersion: "2099-01-01",
		},
		"anthropic": {
			MaxOutputTokens: 4321,
		},
	})
	cfg.refreshEffectiveProviderModels()

	cfg.SetProviderModelConfig("anthropic", ProviderModelConfig{
		MaxOutputTokens: 9999,
	})

	saved := cfg.ProviderModelsForSave()
	if len(saved) != 2 {
		t.Fatalf("len(ProviderModelsForSave()) = %d, want 2", len(saved))
	}

	pm, ok := saved["anthropic"]
	if !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want anthropic entry to be updated in place", saved)
	}
	if pm.DefaultModel != "" {
		t.Fatalf("ProviderModelsForSave()[anthropic].DefaultModel = %q, want empty", pm.DefaultModel)
	}
	if pm.MaxOutputTokens != 9999 {
		t.Fatalf("ProviderModelsForSave()[anthropic].MaxOutputTokens = %d, want %d", pm.MaxOutputTokens, 9999)
	}

	claude, ok := saved["claude"]
	if !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want claude sibling preserved", saved)
	}
	if claude.DefaultModel != "claude-custom" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want %q", claude.DefaultModel, "claude-custom")
	}
	if claude.AnthropicVersion != "2099-01-01" {
		t.Fatalf("ProviderModelsForSave()[claude].AnthropicVersion = %q, want %q", claude.AnthropicVersion, "2099-01-01")
	}
}

func TestPatchProviderModelConfig_PreservesClaudeSiblingWhenUpdatingAnthropicEntry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"claude": {
			DefaultModel:     "claude-custom",
			AnthropicVersion: "2099-01-01",
		},
		"anthropic": {
			DefaultModel:    "anthropic-custom",
			MaxOutputTokens: 4321,
		},
	})
	cfg.refreshEffectiveProviderModels()

	if ok := cfg.PatchProviderModelConfig("anthropic", func(pm *ProviderModelConfig) {
		pm.MaxOutputTokens = 9999
	}); !ok {
		t.Fatal("PatchProviderModelConfig(anthropic) should succeed")
	}

	saved := cfg.ProviderModelsForSave()
	if len(saved) != 2 {
		t.Fatalf("len(ProviderModelsForSave()) = %d, want 2", len(saved))
	}
	if got := cfg.GetSelectedModelForProvider("anthropic"); got != "anthropic-custom" {
		t.Fatalf("GetSelectedModelForProvider(anthropic) = %q, want %q", got, "anthropic-custom")
	}
	if anthropic, ok := saved["anthropic"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want anthropic entry preserved", saved)
	} else {
		if anthropic.DefaultModel != "anthropic-custom" {
			t.Fatalf("ProviderModelsForSave()[anthropic].DefaultModel = %q, want %q", anthropic.DefaultModel, "anthropic-custom")
		}
		if anthropic.MaxOutputTokens != 9999 {
			t.Fatalf("ProviderModelsForSave()[anthropic].MaxOutputTokens = %d, want %d", anthropic.MaxOutputTokens, 9999)
		}
	}
	if claude, ok := saved["claude"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want claude sibling preserved", saved)
	} else if claude.DefaultModel != "claude-custom" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want %q", claude.DefaultModel, "claude-custom")
	}
}

func TestSyncProviderDefaultModel_UpdatesRequestedAnthropicEntryWithoutCollapsingSibling(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"claude": {
			DefaultModel:     "claude-old",
			AnthropicVersion: "2099-01-01",
			MaxOutputTokens:  1234,
		},
		"anthropic": {
			DefaultModel: "anthropic-old",
		},
	})

	if ok := cfg.SyncProviderDefaultModel("anthropic", "claude-new"); !ok {
		t.Fatal("SyncProviderDefaultModel(anthropic, claude-new) should succeed")
	}

	if got := cfg.GetSelectedModelForProvider("anthropic"); got != "claude-new" {
		t.Fatalf("GetSelectedModelForProvider(anthropic) = %q, want %q", got, "claude-new")
	}
	if got := cfg.GetSelectedModelForProvider("claude"); got != "claude-old" {
		t.Fatalf("GetSelectedModelForProvider(claude) = %q, want %q", got, "claude-old")
	}

	saved := cfg.ProviderModelsForSave()
	if len(saved) != 2 {
		t.Fatalf("len(ProviderModelsForSave()) = %d, want 2", len(saved))
	}

	pm, ok := saved["anthropic"]
	if !ok {
		t.Fatal("ProviderModelsForSave() should keep anthropic key after anthropic sync")
	}
	if pm.DefaultModel != "claude-new" {
		t.Fatalf("ProviderModelsForSave()[anthropic].DefaultModel = %q, want %q", pm.DefaultModel, "claude-new")
	}
	if pm.AnthropicVersion != "" {
		t.Fatalf("ProviderModelsForSave()[anthropic].AnthropicVersion = %q, want empty", pm.AnthropicVersion)
	}
	if pm.MaxOutputTokens != 0 {
		t.Fatalf("ProviderModelsForSave()[anthropic].MaxOutputTokens = %d, want %d", pm.MaxOutputTokens, 0)
	}
	if claude, ok := saved["claude"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want claude sibling preserved", saved)
	} else if claude.DefaultModel != "claude-old" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want %q", claude.DefaultModel, "claude-old")
	}
}

func TestSyncProviderDefaultModel_CreatesRequestedAnthropicEntryWhenDefaultProviderIsClaude(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "claude"
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"claude": {
			DefaultModel: "claude-old",
		},
	})

	if ok := cfg.SyncProviderDefaultModel("anthropic", "anthropic-new"); !ok {
		t.Fatal("SyncProviderDefaultModel(anthropic, anthropic-new) should succeed")
	}

	saved := cfg.ProviderModelsForSave()
	if anthropic, ok := saved["anthropic"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want anthropic entry created", saved)
	} else if anthropic.DefaultModel != "anthropic-new" {
		t.Fatalf("ProviderModelsForSave()[anthropic].DefaultModel = %q, want %q", anthropic.DefaultModel, "anthropic-new")
	}
	if claude, ok := saved["claude"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want claude entry preserved", saved)
	} else if claude.DefaultModel != "claude-old" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want %q", claude.DefaultModel, "claude-old")
	}
}

func TestDeleteProviderModelConfig_RemovesOnlyRequestedClaudeEntryWhenAnthropicSiblingExists(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"claude": {
			DefaultModel:     "claude-old",
			AnthropicVersion: "2099-01-01",
		},
		"anthropic": {
			DefaultModel: "anthropic-old",
		},
	})

	cfg.DeleteProviderModelConfig("claude")

	saved := cfg.ProviderModelsForSave()
	if len(saved) != 1 {
		t.Fatalf("ProviderModelsForSave() = %#v, want only anthropic sibling to remain", saved)
	}
	if _, ok := saved["claude"]; ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want claude entry removed", saved)
	}
	if pm, ok := saved["anthropic"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want anthropic sibling preserved", saved)
	} else if pm.DefaultModel != "anthropic-old" {
		t.Fatalf("ProviderModelsForSave()[anthropic].DefaultModel = %q, want %q", pm.DefaultModel, "anthropic-old")
	}

	want := "anthropic-old"
	if got := cfg.GetSelectedModelForProvider("claude"); got != want {
		t.Fatalf("GetSelectedModelForProvider(claude) = %q, want %q", got, want)
	}
	if got := cfg.GetSelectedModelForProvider("anthropic"); got != want {
		t.Fatalf("GetSelectedModelForProvider(anthropic) = %q, want %q", got, want)
	}
}

func TestDeleteProviderModelConfig_NoRawOverrideRestoresEffectiveDefault(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateAbsent, nil)
	cfg.ProviderModels = buildEffectiveProviderModels(nil)

	cfg.ProviderModels["openai"] = mergeProviderModelConfig(cfg.ProviderModels["openai"], ProviderModelConfig{
		DefaultModel: "gpt-custom",
	})
	cfg.DeleteProviderModelConfig("openai")

	want := DefaultConfig().ProviderModels["openai"].DefaultModel
	if got := cfg.ProviderModels["openai"].DefaultModel; got != want {
		t.Fatalf("DeleteProviderModelConfig(openai) fallback default = %q, want %q", got, want)
	}
	if got := cfg.providerModelSectionState(); got != providerModelSectionStateAbsent {
		t.Fatalf("providerModelSectionState() = %v, want %v", got, providerModelSectionStateAbsent)
	}
}

func TestDeleteProviderModelConfig_NoRawOverrideDeletesUnknownEffectiveEntry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateAbsent, nil)
	cfg.ProviderModels["custom-provider"] = ProviderModelConfig{DefaultModel: "custom-model"}

	cfg.DeleteProviderModelConfig("custom-provider")

	if _, ok := cfg.ProviderModels["custom-provider"]; ok {
		t.Fatalf("DeleteProviderModelConfig(custom-provider) should delete unknown effective entry, got %#v", cfg.ProviderModels["custom-provider"])
	}
}

func TestDeleteProviderModelKeys_DirectBranches(t *testing.T) {
	deleteProviderModelKeys(nil, []string{"openai"})

	raw := map[string]ProviderModelConfig{
		"openai":    {DefaultModel: "gpt-custom"},
		"anthropic": {DefaultModel: "claude-custom"},
	}
	deleteProviderModelKeys(raw, []string{"openai", "missing"})

	if _, ok := raw["openai"]; ok {
		t.Fatalf("deleteProviderModelKeys() should delete openai entry, got %#v", raw)
	}
	if got := raw["anthropic"].DefaultModel; got != "claude-custom" {
		t.Fatalf("deleteProviderModelKeys() should preserve unmatched entries, got anthropic=%q", got)
	}
}

func TestProviderModelDeleteActionFor_DirectBranches(t *testing.T) {
	if _, ok := providerModelDeleteActionFor(nil, ""); ok {
		t.Fatal("providerModelDeleteActionFor(empty provider) = ok, want false")
	}

	raw := map[string]ProviderModelConfig{
		"claude": {},
	}
	action, ok := providerModelDeleteActionFor(raw, "anthropic")
	if !ok {
		t.Fatal("providerModelDeleteActionFor(anthropic) should succeed")
	}
	if action.requestedKey != "anthropic" {
		t.Fatalf("requestedKey = %q, want %q", action.requestedKey, "anthropic")
	}
	if len(action.deleteKeys) != 1 || action.deleteKeys[0] != "claude" {
		t.Fatalf("deleteKeys = %v, want [claude]", action.deleteKeys)
	}
}

func TestClearProviderDefaultModelOverride_ClearsOnlyRequestedClaudeEntry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"claude": {
			DefaultModel:     "claude-old",
			AnthropicVersion: "2099-01-01",
		},
		"anthropic": {
			DefaultModel:    "anthropic-old",
			MaxOutputTokens: 1234,
		},
	})

	if ok := cfg.ClearProviderDefaultModelOverride("claude"); !ok {
		t.Fatal("ClearProviderDefaultModelOverride(claude) should succeed")
	}

	saved := cfg.ProviderModelsForSave()
	if len(saved) != 2 {
		t.Fatalf("len(ProviderModelsForSave()) = %d, want 2", len(saved))
	}

	pm, ok := saved["claude"]
	if !ok {
		t.Fatal("ProviderModelsForSave() should keep selected claude key")
	}
	if pm.DefaultModel != "" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want empty", pm.DefaultModel)
	}
	if pm.AnthropicVersion != "2099-01-01" {
		t.Fatalf("ProviderModelsForSave()[claude].AnthropicVersion = %q, want %q", pm.AnthropicVersion, "2099-01-01")
	}
	if pm.MaxOutputTokens != 0 {
		t.Fatalf("ProviderModelsForSave()[claude].MaxOutputTokens = %d, want %d", pm.MaxOutputTokens, 0)
	}
	if anthropic, ok := saved["anthropic"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want anthropic sibling preserved", saved)
	} else if anthropic.DefaultModel != "anthropic-old" {
		t.Fatalf("ProviderModelsForSave()[anthropic].DefaultModel = %q, want %q", anthropic.DefaultModel, "anthropic-old")
	}
}

func TestProviderModelWriteKey_PrefersAliasOverrideOverCanonicalDefault(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.SetProviderModelConfig("anthropic", ProviderModelConfig{DefaultModel: "anthropic-custom"})

	key, ok := cfg.ProviderModelWriteKey("claude")
	if !ok {
		t.Fatal("ProviderModelWriteKey(claude) should succeed")
	}
	if key != "anthropic" {
		t.Fatalf("ProviderModelWriteKey(claude) = %q, want %q", key, "anthropic")
	}
}

func TestProviderModelWriteKey_PrefersDefaultProviderAliasWhenPresent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.SetProviderModelConfig("anthropic", ProviderModelConfig{DefaultModel: "anthropic-custom"})

	key, ok := cfg.ProviderModelWriteKey("claude")
	if !ok {
		t.Fatal("ProviderModelWriteKey(claude) should succeed")
	}
	if key != "anthropic" {
		t.Fatalf("ProviderModelWriteKey(claude) = %q, want %q", key, "anthropic")
	}
}

func TestProviderModelWriteKey_CreatesExactRequestedAliasWhenNoRawEntryExists(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "claude"

	key, ok := cfg.ProviderModelWriteKey("anthropic")
	if !ok {
		t.Fatal("ProviderModelWriteKey(anthropic) should succeed")
	}
	if key != "anthropic" {
		t.Fatalf("ProviderModelWriteKey(anthropic) = %q, want %q", key, "anthropic")
	}
}

func TestProviderModelWriteKey_ExplicitExactKeyWinsAtDefaultValue(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
		"claude": {
			DefaultModel: DefaultConfig().ProviderModels["claude"].DefaultModel,
		},
	})

	key, ok := cfg.ProviderModelWriteKey("claude")
	if !ok {
		t.Fatal("ProviderModelWriteKey(claude) should succeed")
	}
	if key != "claude" {
		t.Fatalf("ProviderModelWriteKey(claude) = %q, want %q", key, "claude")
	}

	if key, ok = cfg.ProviderModelWriteKey("anthropic"); !ok {
		t.Fatal("ProviderModelWriteKey(anthropic) should succeed")
	} else if key != "anthropic" {
		t.Fatalf("ProviderModelWriteKey(anthropic) = %q, want %q", key, "anthropic")
	}
}

func TestUpdateExistingProviderModelConfig_LoadedConfigCreatesProviderEntry(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: openai
default_model: gpt-old
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if ok := cfg.UpdateExistingProviderModelConfig("openai", func(pm *ProviderModelConfig) {
		pm.DefaultModel = "gpt-new"
	}); !ok {
		t.Fatal("UpdateExistingProviderModelConfig(openai) should succeed")
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if got := loaded.GetModelForProvider("openai"); got != "gpt-new" {
		t.Fatalf("GetModelForProvider(openai) = %q, want %q", got, "gpt-new")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "provider_models:") || !strings.Contains(content, "openai:") {
		t.Fatalf("saved config should contain provider_models.openai entry, got:\n%s", content)
	}
}

func TestUpdateExistingProviderModelConfig_FirstRunPersistsProviderEntry(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "gpt-new"

	if ok := cfg.UpdateExistingProviderModelConfig("openai", func(pm *ProviderModelConfig) {
		pm.DefaultModel = "gpt-new"
	}); !ok {
		t.Fatal("UpdateExistingProviderModelConfig(openai) should succeed")
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if got := loaded.GetModelForProvider("openai"); got != "gpt-new" {
		t.Fatalf("GetModelForProvider(openai) = %q, want %q", got, "gpt-new")
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, ".xelyon", "config.yaml"))
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "provider_models:") || !strings.Contains(content, "openai:") {
		t.Fatalf("saved config should contain provider_models.openai entry, got:\n%s", content)
	}
}
