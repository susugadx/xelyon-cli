package config

import "testing"

func TestGetExplicitProviderModelConfig_UsesExplicitAliasAndMergesDefaults(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"anthropic": {
			DefaultModel:    "anthropic-custom",
			MaxOutputTokens: 1234,
		},
	})
	cfg.refreshEffectiveProviderModels()

	pm, ok := cfg.GetExplicitProviderModelConfig("claude")
	if !ok {
		t.Fatal("GetExplicitProviderModelConfig(claude) should succeed via anthropic alias")
	}
	if pm.DefaultModel != "anthropic-custom" {
		t.Fatalf("DefaultModel = %q, want %q", pm.DefaultModel, "anthropic-custom")
	}
	if pm.MaxOutputTokens != 1234 {
		t.Fatalf("MaxOutputTokens = %d, want %d", pm.MaxOutputTokens, 1234)
	}
	if pm.AnthropicVersion != DefaultConfig().ProviderModels["claude"].AnthropicVersion {
		t.Fatalf("AnthropicVersion = %q, want merged default %q", pm.AnthropicVersion, DefaultConfig().ProviderModels["claude"].AnthropicVersion)
	}

	if _, ok := cfg.GetExplicitProviderModelConfig("unknown"); ok {
		t.Fatal("GetExplicitProviderModelConfig(unknown) should fail")
	}
}

func TestFindProviderByDefaultModelAndInferenceHelpers(t *testing.T) {
	cfg := DefaultConfig()
	if got := cfg.FindProviderByDefaultModel(DefaultConfig().ProviderModels["openai"].DefaultModel); got != "openai" {
		t.Fatalf("FindProviderByDefaultModel(openai default) = %q, want %q", got, "openai")
	}
	if got := cfg.FindProviderByDefaultModel("missing-model"); got != "" {
		t.Fatalf("FindProviderByDefaultModel(missing-model) = %q, want empty", got)
	}

	tests := []struct {
		model string
		want  string
	}{
		{model: "gpt-5.4", want: "openai"},
		{model: "codex-mini", want: "openai"},
		{model: "o3-mini", want: "openai"},
		{model: "gemini-3.1-pro-preview-customtools", want: "gemini"},
		{model: "claude-sonnet-4-6", want: "claude"},
		{model: "deepseek-chat", want: "deepseek"},
		{model: "deepseek-v4-flash", want: "deepseek"},
		{model: "global.anthropic.claude-sonnet-4-6", want: "bedrock"},
		{model: "amazon.nova-pro-v1:0", want: "bedrock"},
		{model: "deepseek.r1-v1:0", want: "bedrock"},
		{model: "deepseek.v3.2", want: "bedrock"},
		{model: "us.writer.palmyra-x5-v1:0", want: "bedrock"},
		{model: "google.gemma-3-4b-it", want: "bedrock"},
		{model: "moonshotai.kimi-k2.5", want: "bedrock"},
		{model: "moonshotai.kimi-k2-thinking", want: "bedrock"},
		{model: "anthropic/claude-sonnet-4.6", want: "openrouter"},
		{model: "moonshot.kimi-k2-thinking", want: ""},
		{model: "unknown-model", want: ""},
	}

	for _, tt := range tests {
		if got := InferProviderFromModel(tt.model); got != tt.want {
			t.Fatalf("InferProviderFromModel(%q) = %q, want %q", tt.model, got, tt.want)
		}
	}
}

func TestRuntimeProviderConfigKeyAndResolveProviderForModel_Fallbacks(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
	})
	cfg.refreshEffectiveProviderModels()

	if got := cfg.RuntimeProviderConfigKey("claude", "anthropic-custom"); got != "claude" {
		t.Fatalf("RuntimeProviderConfigKey(%q, %q) = %q, want %q", "claude", "anthropic-custom", got, "claude")
	}
	if got := cfg.RuntimeProviderConfigKey("openai", "unmatched-model"); got != "openai" {
		t.Fatalf("RuntimeProviderConfigKey(%q, %q) = %q, want %q", "openai", "unmatched-model", got, "openai")
	}
	if got := (*Config)(nil).RuntimeProviderConfigKey("", ""); got != "" {
		t.Fatalf("nil RuntimeProviderConfigKey() = %q, want empty", got)
	}

	if got := cfg.ResolveProviderForModel("openai", ""); got != "openai" {
		t.Fatalf("ResolveProviderForModel(current, empty) = %q, want current provider", got)
	}
	if got := cfg.ResolveProviderForModel("openai", "anthropic-custom"); got != "claude" {
		t.Fatalf("ResolveProviderForModel(selected model) = %q, want claude runtime", got)
	}
	if got := cfg.ResolveProviderForModel("openai", "anthropic/claude-sonnet-4.6"); got != "openrouter" {
		t.Fatalf("ResolveProviderForModel(inferred slash model) = %q, want openrouter", got)
	}
	if got := cfg.ResolveProviderForModel("openai", "deepseek.r1-v1:0"); got != "bedrock" {
		t.Fatalf("ResolveProviderForModel(Bedrock DeepSeek ID) = %q, want bedrock", got)
	}
	if got := cfg.ResolveProviderForModel("openai", "deepseek.v3.2"); got != "bedrock" {
		t.Fatalf("ResolveProviderForModel(Bedrock DeepSeek ID) = %q, want bedrock", got)
	}
	if got := cfg.ResolveProviderForModel("openai", "totally-unknown"); got != "openai" {
		t.Fatalf("ResolveProviderForModel(unknown) = %q, want current provider fallback", got)
	}
}

func TestRuntimeProviderConfigKey_CanonicalizesAzureDisplayName(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelConfig("azure", ProviderModelConfig{
		DefaultModel: "corp-gpt55-deployment",
		CatalogModel: "gpt-5.5",
	})

	if got := cfg.RuntimeProviderConfigKey("Azure OpenAI", "corp-gpt55-deployment"); got != "azure" {
		t.Fatalf("RuntimeProviderConfigKey(Azure OpenAI, deployment) = %q, want azure", got)
	}
	if got := cfg.ModelCatalogName("Azure OpenAI", "corp-gpt55-deployment"); got != "gpt-5.5" {
		t.Fatalf("ModelCatalogName(Azure OpenAI, deployment) = %q, want gpt-5.5", got)
	}
}

func TestConfigProviderKeyMethodWrappers(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "anthropic"

	if got := cfg.PreferredProviderConfigKey(""); got != "anthropic" {
		t.Fatalf("PreferredProviderConfigKey(\"\") = %q, want %q", got, "anthropic")
	}
	if got := cfg.PreferredProviderConfigKey("claude"); got != "claude" {
		t.Fatalf("PreferredProviderConfigKey(%q) = %q, want %q", "claude", got, "claude")
	}
	if got := (*Config)(nil).PreferredProviderConfigKey("anthropic"); got != "anthropic" {
		t.Fatalf("nil PreferredProviderConfigKey(%q) = %q, want %q", "anthropic", got, "anthropic")
	}

	if got := cfg.DefaultModelSyncProviderKey("claude", "deepseek"); got != "anthropic" {
		t.Fatalf("DefaultModelSyncProviderKey() = %q, want current default provider anthropic", got)
	}
	if got := (*Config)(nil).DefaultModelSyncProviderKey("claude", ""); got != "claude" {
		t.Fatalf("nil DefaultModelSyncProviderKey() = %q, want session provider key", got)
	}
}
