package config

import "testing"

func TestClearProviderDefaultModelOverrideExact_RemovesOnlyDefaultModel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"openai": {
			DefaultModel:    "gpt-custom",
			MaxOutputTokens: 999,
		},
	})
	cfg.refreshEffectiveProviderModels()

	if ok := cfg.clearProviderDefaultModelOverrideExact("openai"); !ok {
		t.Fatal("clearProviderDefaultModelOverrideExact(openai) should succeed")
	}

	saved := cfg.ProviderModelsForSave()
	pm, ok := saved["openai"]
	if !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want openai entry preserved", saved)
	}
	if pm.DefaultModel != "" {
		t.Fatalf("ProviderModelsForSave()[openai].DefaultModel = %q, want empty", pm.DefaultModel)
	}
	if pm.MaxOutputTokens != 999 {
		t.Fatalf("ProviderModelsForSave()[openai].MaxOutputTokens = %d, want %d", pm.MaxOutputTokens, 999)
	}
}

func TestClearProviderDefaultModelOverrideExact_DeletesZeroEntryAndMethodHelpers(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"openai": {DefaultModel: "gpt-custom"},
	})
	cfg.refreshEffectiveProviderModels()

	if ok := cfg.clearProviderDefaultModelOverrideExact("openai"); !ok {
		t.Fatal("clearProviderDefaultModelOverrideExact(openai) should succeed")
	}
	if got := cfg.ProviderModelsForSave(); got != nil {
		t.Fatalf("ProviderModelsForSave() = %#v, want nil after deleting zero entry", got)
	}

	if !cfg.clearProviderDefaultModelOverrideExact("unknown") {
		t.Fatal("clearProviderDefaultModelOverrideExact(unknown) should be a no-op success")
	}
	if (*Config)(nil).clearProviderDefaultModelOverrideExact("openai") {
		t.Fatal("nil clearProviderDefaultModelOverrideExact should return false")
	}
}
