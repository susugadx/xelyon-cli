package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetFieldValue_ProviderModelsUsesRawEntriesForLoadedConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: deepseek
default_model: deepseek-chat
provider_models:
  anthropic:
    default_model: anthropic-custom
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	val, err := GetFieldValue(cfg, "provider_models")
	if err != nil {
		t.Fatalf("GetFieldValue(provider_models) error = %v", err)
	}

	providerModels, ok := val.(map[string]ProviderModelConfig)
	if !ok {
		t.Fatalf("GetFieldValue(provider_models) returned %T, want map[string]ProviderModelConfig", val)
	}
	if len(providerModels) != 1 {
		t.Fatalf("len(provider_models) = %d, want 1 raw entry", len(providerModels))
	}
	if _, ok := providerModels["anthropic"]; !ok {
		t.Fatal("provider_models should contain anthropic raw entry")
	}
	if _, ok := providerModels["claude"]; ok {
		t.Fatal("provider_models should not expose effective default claude entry in editor view")
	}
}

func TestGetFieldValue_ProviderModelsIsEmptyWhenLoadedConfigHasNoSection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: openai
default_model: gpt-5.4
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	val, err := GetFieldValue(cfg, "provider_models")
	if err != nil {
		t.Fatalf("GetFieldValue(provider_models) error = %v", err)
	}

	providerModels, ok := val.(map[string]ProviderModelConfig)
	if !ok {
		t.Fatalf("GetFieldValue(provider_models) returned %T, want map[string]ProviderModelConfig", val)
	}
	if len(providerModels) != 0 {
		t.Fatalf("len(provider_models) = %d, want 0 when section is absent", len(providerModels))
	}
}

func TestSetFieldValue_ProviderModelsUpdatesRawSaveState(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"openai": {DefaultModel: "old-model"},
	})
	cfg.refreshEffectiveProviderModels()

	newProviderModels := map[string]ProviderModelConfig{
		"openai": {DefaultModel: "new-model"},
	}
	if err := SetFieldValue(cfg, "provider_models", newProviderModels); err != nil {
		t.Fatalf("SetFieldValue(provider_models) error = %v", err)
	}

	saved := cfg.ProviderModelsForSave()
	if got := saved["openai"].DefaultModel; got != "new-model" {
		t.Fatalf("ProviderModelsForSave()[openai].DefaultModel = %q, want %q", got, "new-model")
	}
}

func TestSetFieldValue_ProviderModelsPreservesSeparateClaudeAnthropicAliasEntries(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "anthropic"

	newProviderModels := map[string]ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
		"claude": {
			DefaultModel:     "claude-custom",
			AnthropicVersion: "2099-01-01",
		},
	}
	if err := SetFieldValue(cfg, "provider_models", newProviderModels); err != nil {
		t.Fatalf("SetFieldValue(provider_models) error = %v", err)
	}

	saved := cfg.ProviderModelsForSave()
	if len(saved) != 2 {
		t.Fatalf("len(ProviderModelsForSave()) = %d, want 2", len(saved))
	}

	pm, ok := saved["anthropic"]
	if !ok {
		t.Fatal("ProviderModelsForSave()['anthropic'] missing")
	}
	if pm.DefaultModel != "anthropic-custom" {
		t.Fatalf("ProviderModelsForSave()['anthropic'].DefaultModel = %q, want %q", pm.DefaultModel, "anthropic-custom")
	}
	if pm.AnthropicVersion != "" {
		t.Fatalf("ProviderModelsForSave()['anthropic'].AnthropicVersion = %q, want empty", pm.AnthropicVersion)
	}
	if claudePM, ok := saved["claude"]; !ok {
		t.Fatal("ProviderModelsForSave()['claude'] missing")
	} else if claudePM.DefaultModel != "claude-custom" {
		t.Fatalf("ProviderModelsForSave()['claude'].DefaultModel = %q, want %q", claudePM.DefaultModel, "claude-custom")
	}
	if got := cfg.GetSelectedModelForProvider("claude"); got != "claude-custom" {
		t.Fatalf("GetSelectedModelForProvider(claude) = %q, want %q", got, "claude-custom")
	}
}

func TestBuildConfigRegistry_ProviderModelsDefaultResetsToAbsentSection(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "custom-global-model"

	categories := BuildConfigRegistry(cfg)
	var providerModelsField *ConfigField
	for i := range categories {
		for j := range categories[i].Fields {
			if categories[i].Fields[j].Path == "provider_models" {
				providerModelsField = &categories[i].Fields[j]
				break
			}
		}
	}

	if providerModelsField == nil {
		t.Fatal("provider_models field not found")
	}

	defaultVal, ok := providerModelsField.Default.(map[string]ProviderModelConfig)
	if !ok {
		t.Fatalf("provider_models default has type %T, want map[string]ProviderModelConfig", providerModelsField.Default)
	}
	if defaultVal != nil {
		t.Fatalf("provider_models default = %#v, want nil map to represent absent section", defaultVal)
	}

	if err := SetFieldValue(cfg, "provider_models", defaultVal); err != nil {
		t.Fatalf("SetFieldValue(provider_models, default) error = %v", err)
	}
	if got := cfg.ProviderModelsForSave(); got != nil {
		t.Fatalf("ProviderModelsForSave() = %#v, want nil after reset to default", got)
	}
	if got := cfg.GetSelectedModelForProvider("openai"); got != "custom-global-model" {
		t.Fatalf("GetSelectedModelForProvider(openai) = %q, want %q", got, "custom-global-model")
	}
}

func TestSetFieldValue_ProviderModelsNilResetRemovesExplicitSection(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"openai": {DefaultModel: "gpt-custom"},
	})
	cfg.refreshEffectiveProviderModels()

	if err := SetFieldValue(cfg, "provider_models", map[string]ProviderModelConfig(nil)); err != nil {
		t.Fatalf("SetFieldValue(provider_models, nil) error = %v", err)
	}

	if got := cfg.ProviderModelsForSave(); got != nil {
		t.Fatalf("ProviderModelsForSave() = %#v, want nil after reset", got)
	}
	if got := cfg.providerModelSectionState(); got != providerModelSectionStateAbsent {
		t.Fatalf("providerModelSectionState() = %v, want %v", got, providerModelSectionStateAbsent)
	}
}
