package config

import "testing"

func TestApplyDefaults_FillsAnthropicAliasProviderModelFields(t *testing.T) {
	cfg := &Config{
		ProviderModels: map[string]ProviderModelConfig{
			"anthropic": {DefaultModel: "anthropic-custom"},
		},
	}
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateInMemoryEffectiveOnly, nil)

	applyDefaults(cfg)

	pm := cfg.ProviderModels["anthropic"]
	claudeDefaults := DefaultConfig().ProviderModels["claude"]
	if pm.DefaultModel != "anthropic-custom" {
		t.Fatalf("DefaultModel = %q, want %q", pm.DefaultModel, "anthropic-custom")
	}
	if pm.MaxOutputTokens != claudeDefaults.MaxOutputTokens {
		t.Fatalf("MaxOutputTokens = %d, want %d", pm.MaxOutputTokens, claudeDefaults.MaxOutputTokens)
	}
	if pm.AnthropicVersion != claudeDefaults.AnthropicVersion {
		t.Fatalf("AnthropicVersion = %q, want %q", pm.AnthropicVersion, claudeDefaults.AnthropicVersion)
	}
}
