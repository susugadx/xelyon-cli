package config

import "testing"

func TestGetModelForProvider_LoadedConfigUsesProviderDefaultWhenProviderOverrideMissing(t *testing.T) {
	writeConfigYAMLForTest(t, `
default_provider: deepseek
default_model: global-custom-model
provider_models:
  openai:
    default_model: gpt-custom
`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if got := cfg.GetModelForProvider("openai"); got != "gpt-custom" {
		t.Fatalf("GetModelForProvider(openai) = %q, want %q", got, "gpt-custom")
	}
	want := DefaultConfig().ProviderModels["deepseek"].DefaultModel
	if got := cfg.GetModelForProvider("deepseek"); got != want {
		t.Fatalf("GetModelForProvider(deepseek) = %q, want provider default %q", got, want)
	}
}

func TestGetModelForProvider_LoadedConfigWithoutProviderModelsUsesProviderDefault(t *testing.T) {
	writeConfigYAMLForTest(t, `
default_provider: openai
default_model: deepseek-chat
`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	want := DefaultConfig().ProviderModels["openai"].DefaultModel
	if got := cfg.GetModelForProvider("openai"); got != want {
		t.Fatalf("GetModelForProvider(openai) = %q, want %q", got, want)
	}
	if got := cfg.ResolveModelForProvider("openai"); got != want {
		t.Fatalf("ResolveModelForProvider(openai) = %q, want %q", got, want)
	}
}

func TestGetModelForProvider_LoadedConfigWithNonModelOverrideUsesMergedDefaultModel(t *testing.T) {
	writeConfigYAMLForTest(t, `
default_provider: openai
default_model: deepseek-chat
provider_models:
  openai:
    max_output_tokens: 99999
`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	want := DefaultConfig().ProviderModels["openai"].DefaultModel
	if got := cfg.GetModelForProvider("openai"); got != want {
		t.Fatalf("GetModelForProvider(openai) = %q, want merged default %q", got, want)
	}
	if got := cfg.GetSelectedModelForProvider("openai"); got != want {
		t.Fatalf("GetSelectedModelForProvider(openai) = %q, want %q", got, want)
	}
}

func TestResolveModelForProvider_LoadedConfigUsesProviderDefaultWhenProviderOverrideMissing(t *testing.T) {
	writeConfigYAMLForTest(t, `
default_provider: deepseek
default_model: global-custom-model
provider_models:
  deepseek:
    default_model: deepseek-custom
`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	want := DefaultConfig().ProviderModels["openai"].DefaultModel
	if got := cfg.ResolveModelForProvider("openai"); got != want {
		t.Fatalf("ResolveModelForProvider(openai) = %q, want provider default %q", got, want)
	}
}

func TestLoadConfig_ExplicitCanonicalProviderModelShadowsAlias(t *testing.T) {
	writeConfigYAMLForTest(t, `
default_provider: claude
provider_models:
  anthropic:
    default_model: anthropic-custom
    anthropic_version: 2099-01-01
    anthropic_beta:
      - alias-beta
  claude:
    default_model: claude-sonnet-4-6
`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	pm, ok := cfg.GetProviderModelConfig("claude")
	if !ok {
		t.Fatal("GetProviderModelConfig(claude) should succeed")
	}
	defaultClaude := DefaultConfig().ProviderModels["claude"]
	if pm.DefaultModel != defaultClaude.DefaultModel {
		t.Fatalf("DefaultModel = %q, want %q", pm.DefaultModel, defaultClaude.DefaultModel)
	}
	if pm.AnthropicVersion != defaultClaude.AnthropicVersion {
		t.Fatalf("AnthropicVersion = %q, want %q", pm.AnthropicVersion, defaultClaude.AnthropicVersion)
	}
	if len(pm.AnthropicBeta) != 0 {
		t.Fatalf("AnthropicBeta = %v, want empty", pm.AnthropicBeta)
	}

	key, ok := cfg.ProviderModelWriteKey("claude")
	if !ok {
		t.Fatal("ProviderModelWriteKey(claude) should succeed")
	}
	if key != "claude" {
		t.Fatalf("ProviderModelWriteKey(claude) = %q, want %q", key, "claude")
	}
}

func TestResolveModelForProvider_AnthropicAliasMatchesClaudeDefaultProvider(t *testing.T) {
	writeConfigYAMLForTest(t, `
default_provider: claude
default_model: claude-custom
provider_models: {}
`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if got := cfg.ResolveModelForProvider("anthropic"); got != "claude-custom" {
		t.Fatalf("ResolveModelForProvider(anthropic) = %q, want %q", got, "claude-custom")
	}
}
