package config

import "testing"

func TestProviderModelStoreRawForEdit_DirectBranches(t *testing.T) {
	effective := map[string]ProviderModelConfig{
		"openai": {DefaultModel: "gpt-custom"},
	}

	t.Run("absent returns nil", func(t *testing.T) {
		store := providerModelStore{state: providerModelSectionStateAbsent}
		if got := store.rawForEdit(effective); got != nil {
			t.Fatalf("rawForEdit(absent) = %#v, want nil", got)
		}
	})

	t.Run("explicit empty returns empty map", func(t *testing.T) {
		store := providerModelStore{state: providerModelSectionStateExplicitEmpty}
		got := store.rawForEdit(effective)
		if got == nil || len(got) != 0 {
			t.Fatalf("rawForEdit(explicit empty) = %#v, want empty map", got)
		}
	})

	t.Run("invalid state falls back to nil", func(t *testing.T) {
		store := providerModelStore{state: providerModelSectionState(999)}
		if got := store.rawForEdit(effective); got != nil {
			t.Fatalf("rawForEdit(invalid) = %#v, want nil", got)
		}
	})

	t.Run("explicit entries return cloned raw map", func(t *testing.T) {
		store := providerModelStore{
			state: providerModelSectionStateExplicitEntries,
			raw:   map[string]ProviderModelConfig{"openai": {DefaultModel: "gpt-custom"}},
		}
		got := store.rawForEdit(effective)
		if got["openai"].DefaultModel != "gpt-custom" {
			t.Fatalf("rawForEdit()[openai].DefaultModel = %q, want %q", got["openai"].DefaultModel, "gpt-custom")
		}
		got["openai"] = ProviderModelConfig{DefaultModel: "mutated"}
		if store.raw["openai"].DefaultModel != "gpt-custom" {
			t.Fatalf("store.raw mutated = %#v, want original data preserved", store.raw["openai"])
		}
	})

	t.Run("in-memory effective only returns cloned effective map", func(t *testing.T) {
		store := providerModelStore{state: providerModelSectionStateInMemoryEffectiveOnly}
		got := store.rawForEdit(effective)
		if got["openai"].DefaultModel != "gpt-custom" {
			t.Fatalf("rawForEdit(in-memory)[openai].DefaultModel = %q, want %q", got["openai"].DefaultModel, "gpt-custom")
		}
		got["openai"] = ProviderModelConfig{DefaultModel: "mutated"}
		if effective["openai"].DefaultModel != "gpt-custom" {
			t.Fatalf("effective mutated = %#v, want original data preserved", effective["openai"])
		}
	})
}

func TestRefreshEffectiveProviderModels_DirectBranches(t *testing.T) {
	var nilCfg *Config
	nilCfg.refreshEffectiveProviderModels()

	cfg := DefaultConfig()
	cfg.ProviderModels = nil
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"openai": {DefaultModel: "gpt-custom"},
	})

	cfg.refreshEffectiveProviderModels()
	if got := cfg.ProviderModels["openai"].DefaultModel; got != "gpt-custom" {
		t.Fatalf("refreshEffectiveProviderModels()[openai].DefaultModel = %q, want %q", got, "gpt-custom")
	}
}

func TestProviderModelStoreTransitions_DirectBranches(t *testing.T) {
	t.Run("editing empty entries from in-memory effective requests reset", func(t *testing.T) {
		store := providerModelStore{state: providerModelSectionStateInMemoryEffectiveOnly}
		transition := store.transitionAfterEditingEntries(nil)
		if transition.state != providerModelSectionStateInMemoryEffectiveOnly {
			t.Fatalf("transitionAfterEditingEntries(nil).state = %v, want %v", transition.state, providerModelSectionStateInMemoryEffectiveOnly)
		}
		if !transition.resetInMemoryEffective {
			t.Fatal("transitionAfterEditingEntries(nil).resetInMemoryEffective = false, want true")
		}
		if transition.raw != nil {
			t.Fatalf("transitionAfterEditingEntries(nil).raw = %#v, want nil", transition.raw)
		}
	})

	t.Run("editing entries from explicit empty preserves entries and state intent", func(t *testing.T) {
		store := providerModelStore{state: providerModelSectionStateExplicitEmpty}
		raw := map[string]ProviderModelConfig{
			"openai": {DefaultModel: "gpt-custom"},
		}
		transition := store.transitionAfterEditingEntries(raw)
		if transition.state != providerModelSectionStateExplicitEntriesPreserveEmpty {
			t.Fatalf("transitionAfterEditingEntries(raw).state = %v, want %v", transition.state, providerModelSectionStateExplicitEntriesPreserveEmpty)
		}
		if transition.resetInMemoryEffective {
			t.Fatal("transitionAfterEditingEntries(raw).resetInMemoryEffective = true, want false")
		}
		if got := transition.raw["openai"].DefaultModel; got != "gpt-custom" {
			t.Fatalf("transitionAfterEditingEntries(raw).raw[openai].DefaultModel = %q, want %q", got, "gpt-custom")
		}
	})

	t.Run("raw mutation delete-all keeps explicit empty state contract", func(t *testing.T) {
		store := providerModelStore{state: providerModelSectionStateExplicitEntriesPreserveEmpty}
		transition := store.transitionAfterRawMutation(nil)
		if transition.state != providerModelSectionStateExplicitEmpty {
			t.Fatalf("transitionAfterRawMutation(nil).state = %v, want %v", transition.state, providerModelSectionStateExplicitEmpty)
		}
		if transition.raw != nil {
			t.Fatalf("transitionAfterRawMutation(nil).raw = %#v, want nil", transition.raw)
		}
	})
}
