package config

import "testing"

func TestProviderModelStoreYAMLInputFromRoot_DirectBranches(t *testing.T) {
	t.Run("missing section keeps sectionExists false", func(t *testing.T) {
		input := providerModelStoreYAMLInputFromRoot(
			[]byte("default_provider: openai"),
			map[string]interface{}{"default_provider": "openai"},
		)
		if input.sectionExists {
			t.Fatalf("providerModelStoreYAMLInputFromRoot().sectionExists = %v, want false", input.sectionExists)
		}
		if input.providerRaw != nil {
			t.Fatalf("providerModelStoreYAMLInputFromRoot().providerRaw = %#v, want nil", input.providerRaw)
		}
	})

	t.Run("section exists parses normalized raw map", func(t *testing.T) {
		data := []byte(`
provider_models:
  " OpenAI ":
    default_model: gpt-custom
`)
		input := providerModelStoreYAMLInputFromRoot(data, parseYAMLRootMap(data))
		if !input.sectionExists {
			t.Fatalf("providerModelStoreYAMLInputFromRoot().sectionExists = %v, want true", input.sectionExists)
		}
		if got := input.providerRaw["openai"].DefaultModel; got != "gpt-custom" {
			t.Fatalf("providerModelStoreYAMLInputFromRoot().providerRaw[openai].DefaultModel = %q, want %q", got, "gpt-custom")
		}
	})
}

func TestProviderModelStoreFromYAMLInput_DirectBranches(t *testing.T) {
	t.Run("section missing keeps absent state", func(t *testing.T) {
		store := providerModelStoreFromYAMLInput(providerModelStoreYAMLInput{
			sectionExists: false,
			providerRaw: map[string]ProviderModelConfig{
				"openai": {DefaultModel: "gpt-custom"},
			},
		})
		if store.state != providerModelSectionStateAbsent {
			t.Fatalf("providerModelStoreFromYAMLInput(missing).state = %v, want %v", store.state, providerModelSectionStateAbsent)
		}
		if store.raw != nil {
			t.Fatalf("providerModelStoreFromYAMLInput(missing).raw = %#v, want nil", store.raw)
		}
	})

	t.Run("section exists with empty raw keeps explicit empty", func(t *testing.T) {
		store := providerModelStoreFromYAMLInput(providerModelStoreYAMLInput{
			sectionExists: true,
			providerRaw:   nil,
		})
		if store.state != providerModelSectionStateExplicitEmpty {
			t.Fatalf("providerModelStoreFromYAMLInput(empty).state = %v, want %v", store.state, providerModelSectionStateExplicitEmpty)
		}
		if store.raw == nil || len(store.raw) != 0 {
			t.Fatalf("providerModelStoreFromYAMLInput(empty).raw = %#v, want empty map", store.raw)
		}
	})

	t.Run("section exists with entries keeps explicit entries", func(t *testing.T) {
		store := providerModelStoreFromYAMLInput(providerModelStoreYAMLInput{
			sectionExists: true,
			providerRaw: map[string]ProviderModelConfig{
				"openai": {DefaultModel: "gpt-custom"},
			},
		})
		if store.state != providerModelSectionStateExplicitEntries {
			t.Fatalf("providerModelStoreFromYAMLInput(entries).state = %v, want %v", store.state, providerModelSectionStateExplicitEntries)
		}
		if got := store.raw["openai"].DefaultModel; got != "gpt-custom" {
			t.Fatalf("providerModelStoreFromYAMLInput(entries).raw[openai].DefaultModel = %q, want %q", got, "gpt-custom")
		}
	})
}
