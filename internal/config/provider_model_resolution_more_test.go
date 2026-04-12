package config

import "testing"

func TestProviderModelResolution_NilAndLookupFallbacks(t *testing.T) {
	var nilCfg *Config

	if got := nilCfg.explicitProviderModelSource(); got != nil {
		t.Fatalf("nil explicitProviderModelSource() = %#v, want nil", got)
	}
	if source, key, ok := nilCfg.explicitProviderModelSelection("openai"); ok || source != nil || key != "" {
		t.Fatalf("nil explicitProviderModelSelection() = (%#v, %q, %v), want (nil, empty, false)", source, key, ok)
	}
	if _, ok := nilCfg.rawExplicitProviderModelConfig("openai"); ok {
		t.Fatal("nil rawExplicitProviderModelConfig(openai) should fail")
	}
	if _, ok := nilCfg.selectedProviderModelLookupKey("openai"); ok {
		t.Fatal("nil selectedProviderModelLookupKey(openai) should fail")
	}
	if got := nilCfg.GetProviderDefaultModel("openai"); got != "" {
		t.Fatalf("nil GetProviderDefaultModel(openai) = %q, want empty", got)
	}
	if got := nilCfg.GetEffectiveModelForProvider("openai"); got != "" {
		t.Fatalf("nil GetEffectiveModelForProvider(openai) = %q, want empty", got)
	}
	if got := nilCfg.FindProviderBySelectedModel("gpt-5.4"); got != "" {
		t.Fatalf("nil FindProviderBySelectedModel() = %q, want empty", got)
	}
}

func TestProviderModelResolution_SelectedLookupAndDefaultFallbacks(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEmpty, nil)
	cfg.refreshEffectiveProviderModels()

	if got := cfg.FindProviderBySelectedModel("   "); got != "" {
		t.Fatalf("FindProviderBySelectedModel(blank) = %q, want empty", got)
	}

	key, ok := cfg.selectedProviderModelLookupKey("openai")
	if !ok || key != "openai" {
		t.Fatalf("selectedProviderModelLookupKey(openai) = (%q, %v), want (openai, true)", key, ok)
	}

	wantEffective := DefaultConfig().ProviderModels["openai"].DefaultModel
	if got := cfg.GetEffectiveModelForProvider("openai"); got != wantEffective {
		t.Fatalf("GetEffectiveModelForProvider(openai) = %q, want %q", got, wantEffective)
	}
	if got := cfg.GetProviderDefaultModel("missing"); got != "" {
		t.Fatalf("GetProviderDefaultModel(missing) = %q, want empty", got)
	}
}

func TestResolveProviderForModel_UsesProviderDefaultBeforeNameInference(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.DefaultModel = "deepseek-custom"
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"openai": {DefaultModel: "gpt-selected"},
	})
	cfg.refreshEffectiveProviderModels()

	openAIDefault := DefaultConfig().ProviderModels["openai"].DefaultModel
	if got := cfg.ResolveProviderForModel("claude", openAIDefault); got != "openai" {
		t.Fatalf("ResolveProviderForModel(%q, %q) = %q, want %q", "claude", openAIDefault, got, "openai")
	}
}
