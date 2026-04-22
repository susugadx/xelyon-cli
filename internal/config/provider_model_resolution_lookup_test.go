package config

import "testing"

func TestMergeSelectedProviderModelConfig_UsesBaseWhenOverrideMissing(t *testing.T) {
	got := mergeSelectedProviderModelConfig("openai", ProviderModelConfig{}, false)
	want := DefaultConfig().ProviderModels["openai"]

	if got.DefaultModel != want.DefaultModel {
		t.Fatalf("mergeSelectedProviderModelConfig(openai, no override).DefaultModel = %q, want %q", got.DefaultModel, want.DefaultModel)
	}
	if got.MaxOutputTokens != want.MaxOutputTokens {
		t.Fatalf("mergeSelectedProviderModelConfig(openai, no override).MaxOutputTokens = %d, want %d", got.MaxOutputTokens, want.MaxOutputTokens)
	}
}

func TestSelectedProviderModelConfig_PrefersRawOverride(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"openai": {DefaultModel: "raw-model"},
	})
	cfg.refreshEffectiveProviderModels()
	cfg.ProviderModels["openai"] = mergeProviderModelConfig(cfg.ProviderModels["openai"], ProviderModelConfig{
		DefaultModel: "effective-model",
	})

	got, ok := cfg.selectedProviderModelConfig("openai")
	if !ok {
		t.Fatal("selectedProviderModelConfig(openai) should succeed")
	}
	if got.DefaultModel != "raw-model" {
		t.Fatalf("selectedProviderModelConfig(openai).DefaultModel = %q, want %q", got.DefaultModel, "raw-model")
	}
}

func TestExplicitSelectedProviderModelConfig_ReturnsClone(t *testing.T) {
	source := map[string]ProviderModelConfig{
		"openai": {
			DefaultModel: "gpt-custom",
			ModelOverrides: map[string]ModelOverride{
				"gpt-custom": {MaxOutputTokens: 1234},
			},
		},
	}

	got, ok := explicitSelectedProviderModelConfig(source, "openai")
	if !ok {
		t.Fatal("explicitSelectedProviderModelConfig(openai) should succeed")
	}

	got.ModelOverrides["gpt-custom"] = ModelOverride{MaxOutputTokens: 1}
	if source["openai"].ModelOverrides["gpt-custom"].MaxOutputTokens != 1234 {
		t.Fatalf("explicitSelectedProviderModelConfig(openai) should clone model overrides, source=%#v", source["openai"].ModelOverrides["gpt-custom"])
	}
}
