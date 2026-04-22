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

func TestProviderModelStatePersistsRawEntries(t *testing.T) {
	tests := []struct {
		state providerModelSectionState
		want  bool
	}{
		{state: providerModelSectionStateAbsent, want: false},
		{state: providerModelSectionStateExplicitEmpty, want: false},
		{state: providerModelSectionStateExplicitEntries, want: true},
		{state: providerModelSectionStateExplicitEntriesPreserveEmpty, want: true},
		{state: providerModelSectionStateImplicitEntries, want: true},
		{state: providerModelSectionStateInMemoryEffectiveOnly, want: false},
	}

	for _, tt := range tests {
		if got := providerModelStatePersistsRawEntries(tt.state); got != tt.want {
			t.Fatalf("providerModelStatePersistsRawEntries(%v) = %v, want %v", tt.state, got, tt.want)
		}
	}
}

func TestProviderModelStoreTransition_DirectBranches(t *testing.T) {
	raw := map[string]ProviderModelConfig{
		"openai": {DefaultModel: "gpt-custom"},
	}

	t.Run("in-memory state can request effective reset", func(t *testing.T) {
		transition := providerModelStoreTransition(providerModelSectionStateInMemoryEffectiveOnly, raw, true)
		if transition.state != providerModelSectionStateInMemoryEffectiveOnly {
			t.Fatalf("state = %v, want %v", transition.state, providerModelSectionStateInMemoryEffectiveOnly)
		}
		if !transition.resetInMemoryEffective {
			t.Fatal("resetInMemoryEffective = false, want true")
		}
		if transition.raw != nil {
			t.Fatalf("raw = %#v, want nil", transition.raw)
		}
	})

	t.Run("in-memory state does not reset when reset is disallowed", func(t *testing.T) {
		transition := providerModelStoreTransition(providerModelSectionStateInMemoryEffectiveOnly, raw, false)
		if transition.resetInMemoryEffective {
			t.Fatal("resetInMemoryEffective = true, want false")
		}
		if transition.raw != nil {
			t.Fatalf("raw = %#v, want nil", transition.raw)
		}
	})

	t.Run("raw persists only for states that own raw entries", func(t *testing.T) {
		transition := providerModelStoreTransition(providerModelSectionStateExplicitEntries, raw, true)
		if transition.raw == nil || transition.raw["openai"].DefaultModel != "gpt-custom" {
			t.Fatalf("raw = %#v, want openai entry", transition.raw)
		}

		transition = providerModelStoreTransition(providerModelSectionStateExplicitEmpty, raw, true)
		if transition.raw != nil {
			t.Fatalf("explicit empty transition raw = %#v, want nil", transition.raw)
		}

		transition = providerModelStoreTransition(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{}, true)
		if transition.raw != nil {
			t.Fatalf("empty raw transition = %#v, want nil", transition.raw)
		}
	})
}

func TestProviderModelStateAfterEditingHelpers(t *testing.T) {
	if got := providerModelStateAfterEditingNoEntries(providerModelSectionStateExplicitEntriesPreserveEmpty); got != providerModelSectionStateExplicitEmpty {
		t.Fatalf("providerModelStateAfterEditingNoEntries(preserve-empty) = %v, want %v", got, providerModelSectionStateExplicitEmpty)
	}
	if got := providerModelStateAfterEditingNoEntries(providerModelSectionStateInMemoryEffectiveOnly); got != providerModelSectionStateInMemoryEffectiveOnly {
		t.Fatalf("providerModelStateAfterEditingNoEntries(in-memory) = %v, want %v", got, providerModelSectionStateInMemoryEffectiveOnly)
	}
	if got := providerModelStateAfterEditingNoEntries(providerModelSectionStateExplicitEntries); got != providerModelSectionStateAbsent {
		t.Fatalf("providerModelStateAfterEditingNoEntries(explicit-entries) = %v, want %v", got, providerModelSectionStateAbsent)
	}

	if got := providerModelStateAfterEditingEntries(providerModelSectionStateExplicitEmpty); got != providerModelSectionStateExplicitEntriesPreserveEmpty {
		t.Fatalf("providerModelStateAfterEditingEntries(explicit-empty) = %v, want %v", got, providerModelSectionStateExplicitEntriesPreserveEmpty)
	}
	if got := providerModelStateAfterEditingEntries(providerModelSectionStateExplicitEntries); got != providerModelSectionStateExplicitEntries {
		t.Fatalf("providerModelStateAfterEditingEntries(explicit-entries) = %v, want %v", got, providerModelSectionStateExplicitEntries)
	}
	if got := providerModelStateAfterEditingEntries(providerModelSectionStateAbsent); got != providerModelSectionStateImplicitEntries {
		t.Fatalf("providerModelStateAfterEditingEntries(absent) = %v, want %v", got, providerModelSectionStateImplicitEntries)
	}
}

func TestNormalizeProviderModelsForEdit_DirectBranches(t *testing.T) {
	if got := normalizeProviderModelsForEdit(nil); got != nil {
		t.Fatalf("normalizeProviderModelsForEdit(nil) = %#v, want nil", got)
	}

	src := map[string]ProviderModelConfig{
		" OpenAI ": {DefaultModel: "gpt-custom"},
	}
	got := normalizeProviderModelsForEdit(src)
	if got["openai"].DefaultModel != "gpt-custom" {
		t.Fatalf("normalizeProviderModelsForEdit()[openai].DefaultModel = %q, want %q", got["openai"].DefaultModel, "gpt-custom")
	}
	got["openai"] = ProviderModelConfig{DefaultModel: "mutated"}
	if src[" OpenAI "].DefaultModel != "gpt-custom" {
		t.Fatalf("normalizeProviderModelsForEdit() should clone input map, src = %#v", src)
	}
}

func TestSetProviderModelsForEdit_DirectBranches(t *testing.T) {
	t.Run("nil input resets to absent state", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
			"openai": {DefaultModel: "gpt-custom"},
		})
		cfg.SetProviderModelsForEdit(nil)

		if got := cfg.providerModelSectionState(); got != providerModelSectionStateAbsent {
			t.Fatalf("providerModelSectionState() = %v, want %v", got, providerModelSectionStateAbsent)
		}
		if got := cfg.ProviderModelsForSave(); got != nil {
			t.Fatalf("ProviderModelsForSave() = %#v, want nil", got)
		}
	})

	t.Run("empty map from in-memory state triggers effective reset path", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateInMemoryEffectiveOnly, nil)
		cfg.ProviderModels = buildEffectiveProviderModels(nil)
		cfg.ProviderModels["openai"] = mergeProviderModelConfig(
			DefaultConfig().ProviderModels["openai"],
			ProviderModelConfig{DefaultModel: "gpt-custom"},
		)

		cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{})

		if got := cfg.providerModelSectionState(); got != providerModelSectionStateInMemoryEffectiveOnly {
			t.Fatalf("providerModelSectionState() = %v, want %v", got, providerModelSectionStateInMemoryEffectiveOnly)
		}
		want := DefaultConfig().ProviderModels["openai"].DefaultModel
		if got := cfg.GetEffectiveModelForProvider("openai"); got != want {
			t.Fatalf("GetEffectiveModelForProvider(openai) = %q, want %q", got, want)
		}
	})
}

func TestApplyProviderModelEditTransition_DirectBranches(t *testing.T) {
	t.Run("reset transition restores in-memory effective defaults", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
			"openai": {DefaultModel: "gpt-custom"},
		})
		cfg.refreshEffectiveProviderModels()

		cfg.applyProviderModelEditTransition(providerModelStoreEditTransition{
			state:                  providerModelSectionStateInMemoryEffectiveOnly,
			resetInMemoryEffective: true,
		})

		if got := cfg.providerModelSectionState(); got != providerModelSectionStateInMemoryEffectiveOnly {
			t.Fatalf("providerModelSectionState() = %v, want %v", got, providerModelSectionStateInMemoryEffectiveOnly)
		}
		want := DefaultConfig().ProviderModels["openai"].DefaultModel
		if got := cfg.GetEffectiveModelForProvider("openai"); got != want {
			t.Fatalf("GetEffectiveModelForProvider(openai) = %q, want %q", got, want)
		}
	})

	t.Run("non-reset transition applies explicit state/raw", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.applyProviderModelEditTransition(providerModelStoreEditTransition{
			state: providerModelSectionStateExplicitEntriesPreserveEmpty,
			raw: map[string]ProviderModelConfig{
				"openai": {DefaultModel: "gpt-custom"},
			},
		})

		if got := cfg.providerModelSectionState(); got != providerModelSectionStateExplicitEntriesPreserveEmpty {
			t.Fatalf("providerModelSectionState() = %v, want %v", got, providerModelSectionStateExplicitEntriesPreserveEmpty)
		}
		if got := cfg.ProviderModelsForSave()["openai"].DefaultModel; got != "gpt-custom" {
			t.Fatalf("ProviderModelsForSave()[openai].DefaultModel = %q, want %q", got, "gpt-custom")
		}
	})
}
