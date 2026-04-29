package config

import "testing"

func TestFindProviderBySelectedModel_PrefersDefaultProviderConfigSpelling(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.DefaultModel = "claude-custom"
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEmpty, nil)
	cfg.refreshEffectiveProviderModels()

	if got := cfg.FindProviderBySelectedModel("claude-custom"); got != "anthropic" {
		t.Fatalf("FindProviderBySelectedModel(%q) = %q, want %q", "claude-custom", got, "anthropic")
	}
}

func TestFindProviderBySelectedModel_FindsAnthropicAliasWhenDisplayProvidersExcludeIt(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
		"claude":    {DefaultModel: "claude-custom"},
	})
	cfg.refreshEffectiveProviderModels()

	if got := cfg.FindProviderBySelectedModel("anthropic-custom"); got != "anthropic" {
		t.Fatalf("FindProviderBySelectedModel(%q) = %q, want %q", "anthropic-custom", got, "anthropic")
	}
}

func TestFindProviderBySelectedModel_ContinuesScanningAliasSiblingAfterDefaultAliasMiss(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
		"claude":    {DefaultModel: "corp-c"},
	})
	cfg.refreshEffectiveProviderModels()

	if got := cfg.FindProviderBySelectedModel("corp-c"); got != "claude" {
		t.Fatalf("FindProviderBySelectedModel(%q) = %q, want %q", "corp-c", got, "claude")
	}
	if got := cfg.RuntimeProviderConfigKey("claude", "corp-c"); got != "claude" {
		t.Fatalf("RuntimeProviderConfigKey(%q, %q) = %q, want %q", "claude", "corp-c", got, "claude")
	}
}

func TestRuntimeProviderConfigKey_PrefersRequestedAliasWhenBothAliasEntriesSelectSameModel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"anthropic": {
			DefaultModel:     "shared-custom",
			AnthropicVersion: "2099-01-01",
		},
		"claude": {
			DefaultModel:     "shared-custom",
			AnthropicVersion: "2024-01-01",
		},
	})
	cfg.refreshEffectiveProviderModels()

	if got := cfg.RuntimeProviderConfigKey("anthropic", "shared-custom"); got != "anthropic" {
		t.Fatalf("RuntimeProviderConfigKey(%q, %q) = %q, want %q", "anthropic", "shared-custom", got, "anthropic")
	}
	if got := cfg.RuntimeProviderConfigKey("claude", "shared-custom"); got != "claude" {
		t.Fatalf("RuntimeProviderConfigKey(%q, %q) = %q, want %q", "claude", "shared-custom", got, "claude")
	}
}

func TestResolveProviderForModel_PreservesConfiguredOwnerBeforeNameInference(t *testing.T) {
	tests := []struct {
		name            string
		defaultProvider string
		defaultModel    string
		state           providerModelSectionState
		raw             map[string]ProviderModelConfig
		currentProvider string
		model           string
		want            string
	}{
		{
			name:            "anthropic alias resolves to claude runtime",
			defaultProvider: "anthropic",
			defaultModel:    "claude-custom",
			state:           providerModelSectionStateExplicitEmpty,
			currentProvider: "openai",
			model:           "claude-custom",
			want:            "claude",
		},
		{
			name:            "ollama keeps opaque local model",
			defaultProvider: "ollama",
			defaultModel:    "deepseek-r1:8b",
			state:           providerModelSectionStateAbsent,
			currentProvider: "ollama",
			model:           "deepseek-r1:8b",
			want:            "ollama",
		},
		{
			name:            "groq keeps slash-form custom model",
			defaultProvider: "groq",
			defaultModel:    "moonshotai/kimi-k2-instruct",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"openai": {DefaultModel: "gpt-explicit"},
			},
			currentProvider: "groq",
			model:           "moonshotai/kimi-k2-instruct",
			want:            "groq",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.DefaultProvider = tt.defaultProvider
			cfg.DefaultModel = tt.defaultModel
			cfg.providerModelsStore = normalizeProviderModelStore(tt.state, tt.raw)
			cfg.refreshEffectiveProviderModels()

			if got := cfg.ResolveProviderForModel(tt.currentProvider, tt.model); got != tt.want {
				t.Fatalf("ResolveProviderForModel(%q, %q) = %q, want %q", tt.currentProvider, tt.model, got, tt.want)
			}
		})
	}
}

func TestResolveProviderForModel_PrefersCurrentProviderSelectedModelWhenNamesCollide(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.SetProviderModelConfig("azure", ProviderModelConfig{
		DefaultModel: "gpt-5.4",
		CatalogModel: "gpt-5.4",
	})

	if got := cfg.ResolveProviderForModel("azure", "gpt-5.4"); got != "azure" {
		t.Fatalf("ResolveProviderForModel(%q, %q) = %q, want %q", "azure", "gpt-5.4", got, "azure")
	}
}

func TestGetExplicitProviderDefaultModel_ExcludesBuiltInProviderDefault(t *testing.T) {
	cfg := DefaultConfig()
	if got := cfg.GetExplicitProviderDefaultModel("azure"); got != "" {
		t.Fatalf("GetExplicitProviderDefaultModel(azure) = %q, want empty without explicit provider_models entry", got)
	}

	cfg.SetProviderModelConfig("Azure OpenAI", ProviderModelConfig{
		DefaultModel: "corp-gpt55-deployment",
		CatalogModel: "gpt-5.5",
	})

	if got := cfg.GetExplicitProviderDefaultModel("azure"); got != "corp-gpt55-deployment" {
		t.Fatalf("GetExplicitProviderDefaultModel(azure) = %q, want explicit Azure deployment", got)
	}
}

func TestGetProviderModelConfig_AliasOverridesCanonicalDefaults(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelConfig("anthropic", ProviderModelConfig{
		DefaultModel:    "anthropic-custom",
		MaxOutputTokens: 12345,
		ModelOverrides: map[string]ModelOverride{
			"anthropic-custom": {MaxOutputTokens: 4321},
		},
	})

	pm, ok := cfg.GetProviderModelConfig("claude")
	if !ok {
		t.Fatal("GetProviderModelConfig(claude) should find anthropic alias config")
	}
	if pm.DefaultModel != "anthropic-custom" {
		t.Fatalf("DefaultModel = %q, want %q", pm.DefaultModel, "anthropic-custom")
	}
	if pm.MaxOutputTokens != 12345 {
		t.Fatalf("MaxOutputTokens = %d, want %d", pm.MaxOutputTokens, 12345)
	}
	if pm.AnthropicVersion != DefaultConfig().ProviderModels["claude"].AnthropicVersion {
		t.Fatalf("AnthropicVersion = %q, want claude default %q", pm.AnthropicVersion, DefaultConfig().ProviderModels["claude"].AnthropicVersion)
	}
	if got := pm.ModelOverrides["anthropic-custom"].MaxOutputTokens; got != 4321 {
		t.Fatalf("ModelOverrides[anthropic-custom].MaxOutputTokens = %d, want %d", got, 4321)
	}
}

func TestGetProviderModelConfig_PrefersRequestedAliasFieldsWhenBothAliasKeysExist(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"anthropic": {
			DefaultModel:     "anthropic-custom",
			AnthropicVersion: "2099-01-01",
			AnthropicBeta:    []string{"alias-beta"},
		},
		"claude": {
			DefaultModel: DefaultConfig().ProviderModels["claude"].DefaultModel,
		},
	})
	cfg.refreshEffectiveProviderModels()

	pm, ok := cfg.GetProviderModelConfig("claude")
	if !ok {
		t.Fatal("GetProviderModelConfig(claude) should succeed")
	}
	if pm.DefaultModel != DefaultConfig().ProviderModels["claude"].DefaultModel {
		t.Fatalf("DefaultModel = %q, want default claude model %q", pm.DefaultModel, DefaultConfig().ProviderModels["claude"].DefaultModel)
	}
	if pm.AnthropicVersion != DefaultConfig().ProviderModels["claude"].AnthropicVersion {
		t.Fatalf("AnthropicVersion = %q, want default claude version %q", pm.AnthropicVersion, DefaultConfig().ProviderModels["claude"].AnthropicVersion)
	}
	if len(pm.AnthropicBeta) != 0 {
		t.Fatalf("AnthropicBeta = %v, want empty when exact claude key is explicit", pm.AnthropicBeta)
	}

	if pmAnthropic, ok := cfg.GetProviderModelConfig("anthropic"); !ok {
		t.Fatal("GetProviderModelConfig(anthropic) should succeed")
	} else {
		if pmAnthropic.DefaultModel != "anthropic-custom" {
			t.Fatalf("GetProviderModelConfig(anthropic).DefaultModel = %q, want %q", pmAnthropic.DefaultModel, "anthropic-custom")
		}
		if pmAnthropic.AnthropicVersion != "2099-01-01" {
			t.Fatalf("GetProviderModelConfig(anthropic).AnthropicVersion = %q, want %q", pmAnthropic.AnthropicVersion, "2099-01-01")
		}
		if len(pmAnthropic.AnthropicBeta) != 1 || pmAnthropic.AnthropicBeta[0] != "alias-beta" {
			t.Fatalf("GetProviderModelConfig(anthropic).AnthropicBeta = %v, want [alias-beta]", pmAnthropic.AnthropicBeta)
		}
	}
}

func TestGetProviderModelConfig_PrefersDefaultProviderAliasSpellingWhenBothAliasKeysExist(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
		"claude":    {DefaultModel: "claude-custom"},
	})

	if got := cfg.GetSelectedModelForProvider("claude"); got != "claude-custom" {
		t.Fatalf("GetSelectedModelForProvider(claude) = %q, want %q", got, "claude-custom")
	}
	if got := cfg.GetSelectedModelForProvider("anthropic"); got != "anthropic-custom" {
		t.Fatalf("GetSelectedModelForProvider(anthropic) = %q, want %q", got, "anthropic-custom")
	}
	if key, ok := cfg.ProviderModelWriteKey("claude"); !ok {
		t.Fatal("ProviderModelWriteKey(claude) should succeed")
	} else if key != "claude" {
		t.Fatalf("ProviderModelWriteKey(claude) = %q, want %q", key, "claude")
	}
}

func TestGetSelectedModelForProvider_PrefersRequestedAnthropicAliasOverCanonicalWhenDefaultProviderUnrelated(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
		"claude":    {DefaultModel: "claude-custom"},
	})
	cfg.refreshEffectiveProviderModels()

	if got := cfg.GetSelectedModelForProvider("anthropic"); got != "anthropic-custom" {
		t.Fatalf("GetSelectedModelForProvider(anthropic) = %q, want %q", got, "anthropic-custom")
	}
	if got := cfg.GetSelectedModelForProvider("claude"); got != "claude-custom" {
		t.Fatalf("GetSelectedModelForProvider(claude) = %q, want %q", got, "claude-custom")
	}
}
