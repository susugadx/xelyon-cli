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

func TestCloneProviderModelConfigMap_NormalizedKeyCollisionUsesStableOwnerPrecedence(t *testing.T) {
	src := map[string]ProviderModelConfig{
		"Azure OpenAI": {
			DefaultModel: "display-deployment",
			CatalogModel: "gpt-5.5",
		},
		"azure": {
			DefaultModel:    "owner-deployment",
			MaxOutputTokens: 12345,
		},
	}

	for i := 0; i < 200; i++ {
		got := cloneProviderModelConfigMap(src)
		pm, ok := got["azure"]
		if !ok {
			t.Fatalf("cloneProviderModelConfigMap() missing azure key: %#v", got)
		}
		if pm.DefaultModel != "owner-deployment" {
			t.Fatalf("DefaultModel = %q, want owner key value", pm.DefaultModel)
		}
		if pm.CatalogModel != "gpt-5.5" {
			t.Fatalf("CatalogModel = %q, want merged display-name value", pm.CatalogModel)
		}
		if pm.MaxOutputTokens != 12345 {
			t.Fatalf("MaxOutputTokens = %d, want owner key value", pm.MaxOutputTokens)
		}
		if _, exists := got["azure openai"]; exists {
			t.Fatalf("unexpected display-name normalized key remains: %#v", got)
		}
	}
}

func TestProviderModelStoreFromYAMLWithRoot_StateResolution(t *testing.T) {
	t.Run("provider_models key missing keeps absent state", func(t *testing.T) {
		data := []byte("default_provider: openai")
		store := providerModelStoreFromYAMLWithRoot(data, map[string]interface{}{
			"default_provider": "openai",
		})
		if store.state != providerModelSectionStateAbsent {
			t.Fatalf("providerModelStoreFromYAMLWithRoot(missing).state = %v, want %v", store.state, providerModelSectionStateAbsent)
		}
		if store.raw != nil {
			t.Fatalf("providerModelStoreFromYAMLWithRoot(missing).raw = %#v, want nil", store.raw)
		}
	})

	t.Run("explicit empty map keeps explicit-empty state", func(t *testing.T) {
		data := []byte("provider_models: {}")
		raw := parseYAMLRootMap(data)
		store := providerModelStoreFromYAMLWithRoot(data, raw)
		if store.state != providerModelSectionStateExplicitEmpty {
			t.Fatalf("providerModelStoreFromYAMLWithRoot(empty).state = %v, want %v", store.state, providerModelSectionStateExplicitEmpty)
		}
		if store.raw == nil || len(store.raw) != 0 {
			t.Fatalf("providerModelStoreFromYAMLWithRoot(empty).raw = %#v, want empty map", store.raw)
		}
	})

	t.Run("entry map becomes explicit entries with normalized key", func(t *testing.T) {
		data := []byte(`
provider_models:
  " OpenAI ":
    default_model: gpt-custom
`)
		raw := parseYAMLRootMap(data)
		store := providerModelStoreFromYAMLWithRoot(data, raw)
		if store.state != providerModelSectionStateExplicitEntries {
			t.Fatalf("providerModelStoreFromYAMLWithRoot(entries).state = %v, want %v", store.state, providerModelSectionStateExplicitEntries)
		}
		if got := store.raw["openai"].DefaultModel; got != "gpt-custom" {
			t.Fatalf("providerModelStoreFromYAMLWithRoot(entries).raw[openai].DefaultModel = %q, want %q", got, "gpt-custom")
		}
	})

	t.Run("invalid provider_models shape keeps explicit-empty contract", func(t *testing.T) {
		store := providerModelStoreFromYAMLWithRoot([]byte("provider_models: [unexpected]"), map[string]interface{}{
			"provider_models": []interface{}{"unexpected"},
		})
		if store.state != providerModelSectionStateExplicitEmpty {
			t.Fatalf("providerModelStoreFromYAMLWithRoot(invalid).state = %v, want %v", store.state, providerModelSectionStateExplicitEmpty)
		}
		if store.raw == nil || len(store.raw) != 0 {
			t.Fatalf("providerModelStoreFromYAMLWithRoot(invalid).raw = %#v, want empty map", store.raw)
		}
	})

	t.Run("anthropic_version date scalar keeps plain date string", func(t *testing.T) {
		data := []byte(`
provider_models:
  anthropic:
    anthropic_version: 2099-01-01
`)
		raw := parseYAMLRootMap(data)
		store := providerModelStoreFromYAMLWithRoot(data, raw)
		if store.state != providerModelSectionStateExplicitEntries {
			t.Fatalf("providerModelStoreFromYAMLWithRoot(date).state = %v, want %v", store.state, providerModelSectionStateExplicitEntries)
		}
		if got := store.raw["anthropic"].AnthropicVersion; got != "2099-01-01" {
			t.Fatalf("providerModelStoreFromYAMLWithRoot(date).raw[anthropic].AnthropicVersion = %q, want %q", got, "2099-01-01")
		}
	})
}

func TestProviderModelStoreStateFromParsedYAML_DirectBranches(t *testing.T) {
	t.Run("empty raw keeps explicit-empty contract", func(t *testing.T) {
		store := providerModelStoreStateFromParsedYAML(nil)
		if store.state != providerModelSectionStateExplicitEmpty {
			t.Fatalf("providerModelStoreStateFromParsedYAML(nil).state = %v, want %v", store.state, providerModelSectionStateExplicitEmpty)
		}
		if store.raw == nil || len(store.raw) != 0 {
			t.Fatalf("providerModelStoreStateFromParsedYAML(nil).raw = %#v, want empty map", store.raw)
		}
	})

	t.Run("non-empty raw keeps explicit-entries contract", func(t *testing.T) {
		store := providerModelStoreStateFromParsedYAML(map[string]ProviderModelConfig{
			"openai": {DefaultModel: "gpt-custom"},
		})
		if store.state != providerModelSectionStateExplicitEntries {
			t.Fatalf("providerModelStoreStateFromParsedYAML(entries).state = %v, want %v", store.state, providerModelSectionStateExplicitEntries)
		}
		if got := store.raw["openai"].DefaultModel; got != "gpt-custom" {
			t.Fatalf("providerModelStoreStateFromParsedYAML(entries).raw[openai].DefaultModel = %q, want %q", got, "gpt-custom")
		}
	})
}

func TestProviderModelStoreStateFromYAMLSection_DirectBranches(t *testing.T) {
	if got := providerModelsSectionExists(nil); got {
		t.Fatal("providerModelsSectionExists(nil) = true, want false")
	}
	if got := providerModelsSectionExists(map[string]interface{}{"provider_models": map[string]interface{}{}}); !got {
		t.Fatal("providerModelsSectionExists(provider_models) = false, want true")
	}

	store := providerModelStoreStateFromYAMLSection(false, map[string]ProviderModelConfig{
		"openai": {DefaultModel: "gpt-custom"},
	})
	if store.state != providerModelSectionStateAbsent {
		t.Fatalf("providerModelStoreStateFromYAMLSection(false, entries).state = %v, want %v", store.state, providerModelSectionStateAbsent)
	}
	if store.raw != nil {
		t.Fatalf("providerModelStoreStateFromYAMLSection(false, entries).raw = %#v, want nil", store.raw)
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
