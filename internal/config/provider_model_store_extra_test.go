package config

import "testing"

func TestProviderModelStore_StateAfterEditingEntries(t *testing.T) {
	tests := []struct {
		name       string
		state      providerModelSectionState
		entryCount int
		want       providerModelSectionState
	}{
		{name: "absent empty stays absent", state: providerModelSectionStateAbsent, entryCount: 0, want: providerModelSectionStateAbsent},
		{name: "absent with entries becomes implicit", state: providerModelSectionStateAbsent, entryCount: 1, want: providerModelSectionStateImplicitEntries},
		{name: "explicit empty stays explicit empty", state: providerModelSectionStateExplicitEmpty, entryCount: 0, want: providerModelSectionStateExplicitEmpty},
		{name: "explicit empty with entries preserves emptiness intent", state: providerModelSectionStateExplicitEmpty, entryCount: 1, want: providerModelSectionStateExplicitEntriesPreserveEmpty},
		{name: "explicit entries empty becomes absent", state: providerModelSectionStateExplicitEntries, entryCount: 0, want: providerModelSectionStateAbsent},
		{name: "explicit entries stay explicit entries", state: providerModelSectionStateExplicitEntries, entryCount: 2, want: providerModelSectionStateExplicitEntries},
		{name: "preserve empty with zero returns explicit empty", state: providerModelSectionStateExplicitEntriesPreserveEmpty, entryCount: 0, want: providerModelSectionStateExplicitEmpty},
		{name: "preserve empty with entries stays preserve empty", state: providerModelSectionStateExplicitEntriesPreserveEmpty, entryCount: 2, want: providerModelSectionStateExplicitEntriesPreserveEmpty},
		{name: "implicit empty becomes absent", state: providerModelSectionStateImplicitEntries, entryCount: 0, want: providerModelSectionStateAbsent},
		{name: "implicit with entries stays implicit", state: providerModelSectionStateImplicitEntries, entryCount: 2, want: providerModelSectionStateImplicitEntries},
		{name: "in-memory empty stays in-memory", state: providerModelSectionStateInMemoryEffectiveOnly, entryCount: 0, want: providerModelSectionStateInMemoryEffectiveOnly},
		{name: "in-memory with entries becomes implicit", state: providerModelSectionStateInMemoryEffectiveOnly, entryCount: 1, want: providerModelSectionStateImplicitEntries},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := providerModelStore{state: tt.state}
			if got := store.stateAfterEditingEntries(tt.entryCount); got != tt.want {
				t.Fatalf("stateAfterEditingEntries(%d) = %v, want %v", tt.entryCount, got, tt.want)
			}
		})
	}
}

func TestNormalizeProviderModelStore_StateNormalization(t *testing.T) {
	tests := []struct {
		name      string
		state     providerModelSectionState
		raw       map[string]ProviderModelConfig
		wantState providerModelSectionState
		wantNil   bool
		wantLen   int
	}{
		{
			name:      "invalid state falls back to absent",
			state:     providerModelSectionState(999),
			raw:       map[string]ProviderModelConfig{"openai": {DefaultModel: "gpt-custom"}},
			wantState: providerModelSectionStateAbsent,
			wantNil:   true,
		},
		{
			name:      "implicit entries with no raw becomes absent",
			state:     providerModelSectionStateImplicitEntries,
			raw:       nil,
			wantState: providerModelSectionStateAbsent,
			wantNil:   true,
		},
		{
			name:      "explicit entries with no raw becomes explicit empty",
			state:     providerModelSectionStateExplicitEntries,
			raw:       nil,
			wantState: providerModelSectionStateExplicitEmpty,
			wantNil:   false,
			wantLen:   0,
		},
		{
			name:      "in-memory ignores raw source",
			state:     providerModelSectionStateInMemoryEffectiveOnly,
			raw:       map[string]ProviderModelConfig{"openai": {DefaultModel: "gpt-custom"}},
			wantState: providerModelSectionStateInMemoryEffectiveOnly,
			wantNil:   true,
		},
		{
			name:      "normalizes provider names",
			state:     providerModelSectionStateExplicitEntries,
			raw:       map[string]ProviderModelConfig{" OpenAI ": {DefaultModel: "gpt-custom"}},
			wantState: providerModelSectionStateExplicitEntries,
			wantNil:   false,
			wantLen:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeProviderModelStore(tt.state, tt.raw)
			if got.state != tt.wantState {
				t.Fatalf("normalizeProviderModelStore().state = %v, want %v", got.state, tt.wantState)
			}
			if (got.raw == nil) != tt.wantNil {
				t.Fatalf("normalizeProviderModelStore().raw nil = %v, want %v", got.raw == nil, tt.wantNil)
			}
			if !tt.wantNil && len(got.raw) != tt.wantLen {
				t.Fatalf("len(normalizeProviderModelStore().raw) = %d, want %d", len(got.raw), tt.wantLen)
			}
			if tt.name == "normalizes provider names" {
				if _, ok := got.raw["openai"]; !ok {
					t.Fatalf("normalized raw = %#v, want openai key", got.raw)
				}
			}
		})
	}
}

func TestConfig_EffectiveProviderModelMutators(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProviderModels = nil

	cfg.setEffectiveProviderModelConfig("custom", ProviderModelConfig{DefaultModel: "custom-model", MaxOutputTokens: 42})
	pm, ok := cfg.effectiveProviderModelConfig("custom")
	if !ok {
		t.Fatal("effectiveProviderModelConfig(custom) should succeed")
	}
	if pm.DefaultModel != "custom-model" || pm.MaxOutputTokens != 42 {
		t.Fatalf("effectiveProviderModelConfig(custom) = %#v, want custom-model/42", pm)
	}

	cfg.deleteEffectiveProviderModelConfig("custom")
	if _, ok := cfg.effectiveProviderModelConfig("custom"); ok {
		t.Fatal("effectiveProviderModelConfig(custom) should be deleted")
	}

	cfg.SetProviderModelConfig("openai", ProviderModelConfig{DefaultModel: "gpt-custom"})
	cfg.resetInMemoryEffectiveProviderModels()
	if got := cfg.providerModelSectionState(); got != providerModelSectionStateInMemoryEffectiveOnly {
		t.Fatalf("providerModelSectionState() = %v, want %v", got, providerModelSectionStateInMemoryEffectiveOnly)
	}
	want := DefaultConfig().ProviderModels["openai"].DefaultModel
	if got := cfg.GetProviderDefaultModel("openai"); got != want {
		t.Fatalf("GetProviderDefaultModel(openai) = %q, want %q after reset", got, want)
	}
	if cfg.ProviderModelsForSave() != nil {
		t.Fatalf("ProviderModelsForSave() = %#v, want nil after reset", cfg.ProviderModelsForSave())
	}
}
