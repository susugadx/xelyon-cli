package config

import "testing"

func TestExtractRawProviderModelsFromYAML_InvalidAndNormalizedKeys(t *testing.T) {
	if got := extractRawProviderModelsFromYAML([]byte("provider_models: [")); got != nil {
		t.Fatalf("extractRawProviderModelsFromYAML(invalid) = %#v, want nil", got)
	}

	data := []byte(`
provider_models:
  " OpenAI ":
    default_model: gpt-custom
  "   ":
    default_model: ignored
`)
	got := extractRawProviderModelsFromYAML(data)
	if len(got) != 1 {
		t.Fatalf("len(extractRawProviderModelsFromYAML()) = %d, want 1", len(got))
	}
	if got["openai"].DefaultModel != "gpt-custom" {
		t.Fatalf("extractRawProviderModelsFromYAML()[openai].DefaultModel = %q, want %q", got["openai"].DefaultModel, "gpt-custom")
	}
}

func TestDiffProviderModelConfig_TracksOnlyEffectiveChanges(t *testing.T) {
	base := ProviderModelConfig{
		DefaultModel:     "gpt-5.4",
		MaxOutputTokens:  4096,
		AnthropicVersion: "2024-01-01",
		AnthropicBeta:    []string{"beta-a"},
		ModelOverrides: map[string]ModelOverride{
			"gpt-5.4": {MaxOutputTokens: 1024},
		},
	}

	t.Run("equal and zero-like fields collapse to empty diff", func(t *testing.T) {
		current := ProviderModelConfig{
			DefaultModel:     "gpt-5.4",
			AnthropicVersion: "2024-01-01",
			AnthropicBeta:    []string{"beta-a"},
			ModelOverrides: map[string]ModelOverride{
				"gpt-5.4": {MaxOutputTokens: 1024},
			},
		}

		if got := diffProviderModelConfig(base, current); !isZeroProviderModelConfig(got) {
			t.Fatalf("diffProviderModelConfig(equal) = %#v, want zero diff", got)
		}
	})

	t.Run("changed fields are copied into diff", func(t *testing.T) {
		current := ProviderModelConfig{
			DefaultModel:     "gpt-custom",
			MaxOutputTokens:  2048,
			AnthropicVersion: "2099-01-01",
			AnthropicBeta:    []string{"beta-b"},
			ModelOverrides: map[string]ModelOverride{
				"gpt-custom": {MaxOutputTokens: 512},
			},
		}

		got := diffProviderModelConfig(base, current)
		if got.DefaultModel != "gpt-custom" {
			t.Fatalf("DefaultModel = %q, want %q", got.DefaultModel, "gpt-custom")
		}
		if got.MaxOutputTokens != 2048 {
			t.Fatalf("MaxOutputTokens = %d, want %d", got.MaxOutputTokens, 2048)
		}
		if got.AnthropicVersion != "2099-01-01" {
			t.Fatalf("AnthropicVersion = %q, want %q", got.AnthropicVersion, "2099-01-01")
		}
		if len(got.AnthropicBeta) != 1 || got.AnthropicBeta[0] != "beta-b" {
			t.Fatalf("AnthropicBeta = %v, want [beta-b]", got.AnthropicBeta)
		}
		if got.ModelOverrides["gpt-custom"].MaxOutputTokens != 512 {
			t.Fatalf("ModelOverrides[gpt-custom].MaxOutputTokens = %d, want %d", got.ModelOverrides["gpt-custom"].MaxOutputTokens, 512)
		}
	})
}

func TestConfigProviderModelStore_InMemorySourcesAndMutationHelpers(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateInMemoryEffectiveOnly, nil)
	cfg.ProviderModels = buildEffectiveProviderModels(nil)

	customOpenAI := mergeProviderModelConfig(
		DefaultConfig().ProviderModels["openai"],
		ProviderModelConfig{DefaultModel: "gpt-custom"},
	)
	cfg.ProviderModels["openai"] = customOpenAI

	explicit := cfg.explicitProviderModelSource()
	if explicit["openai"].DefaultModel != "gpt-custom" {
		t.Fatalf("explicitProviderModelSource()[openai].DefaultModel = %q, want %q", explicit["openai"].DefaultModel, "gpt-custom")
	}
	if _, ok := explicit["deepseek"]; ok {
		t.Fatalf("explicitProviderModelSource() = %#v, want only in-memory diff entries", explicit)
	}

	refreshSource := cfg.effectiveProviderModelRefreshSource()
	if refreshSource["openai"].DefaultModel != "gpt-custom" {
		t.Fatalf("effectiveProviderModelRefreshSource()[openai].DefaultModel = %q, want %q", refreshSource["openai"].DefaultModel, "gpt-custom")
	}

	edit := cfg.ProviderModelsForEdit()
	if edit["openai"].DefaultModel != "gpt-custom" {
		t.Fatalf("ProviderModelsForEdit()[openai].DefaultModel = %q, want %q", edit["openai"].DefaultModel, "gpt-custom")
	}
	edit["openai"] = ProviderModelConfig{DefaultModel: "mutated"}
	if cfg.ProviderModelsForEdit()["openai"].DefaultModel != "gpt-custom" {
		t.Fatalf("ProviderModelsForEdit() should return cloned data, got %#v", cfg.ProviderModelsForEdit()["openai"])
	}

	clonedRaw := cfg.clonedRawProviderModelsForMutation()
	if clonedRaw["openai"].DefaultModel != "gpt-custom" {
		t.Fatalf("clonedRawProviderModelsForMutation()[openai].DefaultModel = %q, want %q", clonedRaw["openai"].DefaultModel, "gpt-custom")
	}

	mutableRaw := cfg.mutableRawProviderModelsForMutation()
	delete(mutableRaw, "openai")
	if cfg.mutableRawProviderModelsForMutation()["openai"].DefaultModel != "gpt-custom" {
		t.Fatalf("mutableRawProviderModelsForMutation() should return isolated map, got %#v", cfg.mutableRawProviderModelsForMutation()["openai"])
	}

	cfg.applyRawProviderModelMutation(nil)
	if got := cfg.providerModelSectionState(); got != providerModelSectionStateAbsent {
		t.Fatalf("providerModelSectionState() after applyRawProviderModelMutation(nil) = %v, want %v", got, providerModelSectionStateAbsent)
	}
	wantDefault := DefaultConfig().ProviderModels["openai"].DefaultModel
	if got := cfg.GetEffectiveModelForProvider("openai"); got != wantDefault {
		t.Fatalf("GetEffectiveModelForProvider(openai) = %q, want %q after reset", got, wantDefault)
	}
}

func TestConfigProviderModelStore_NilWrappers(t *testing.T) {
	var cfg *Config

	if got := cfg.ProviderModelsForSave(); got != nil {
		t.Fatalf("nil ProviderModelsForSave() = %#v, want nil", got)
	}
	if got := cfg.ProviderModelsForEdit(); got != nil {
		t.Fatalf("nil ProviderModelsForEdit() = %#v, want nil", got)
	}
	if got := cfg.effectiveProviderModels(); got != nil {
		t.Fatalf("nil effectiveProviderModels() = %#v, want nil", got)
	}
	if got := cfg.effectiveProviderModelRefreshSource(); got != nil {
		t.Fatalf("nil effectiveProviderModelRefreshSource() = %#v, want nil", got)
	}
}
