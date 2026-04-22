package config

import "testing"

func TestDeleteProviderModelConfig_RemovesOnlyRequestedClaudeEntryWhenAnthropicSiblingExists(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"claude": {
			DefaultModel:     "claude-old",
			AnthropicVersion: "2099-01-01",
		},
		"anthropic": {
			DefaultModel: "anthropic-old",
		},
	})

	cfg.DeleteProviderModelConfig("claude")

	saved := cfg.ProviderModelsForSave()
	if len(saved) != 1 {
		t.Fatalf("ProviderModelsForSave() = %#v, want only anthropic sibling to remain", saved)
	}
	if _, ok := saved["claude"]; ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want claude entry removed", saved)
	}
	if pm, ok := saved["anthropic"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want anthropic sibling preserved", saved)
	} else if pm.DefaultModel != "anthropic-old" {
		t.Fatalf("ProviderModelsForSave()[anthropic].DefaultModel = %q, want %q", pm.DefaultModel, "anthropic-old")
	}

	want := "anthropic-old"
	if got := cfg.GetSelectedModelForProvider("claude"); got != want {
		t.Fatalf("GetSelectedModelForProvider(claude) = %q, want %q", got, want)
	}
	if got := cfg.GetSelectedModelForProvider("anthropic"); got != want {
		t.Fatalf("GetSelectedModelForProvider(anthropic) = %q, want %q", got, want)
	}
}

func TestDeleteProviderModelConfig_NoRawOverrideRestoresEffectiveDefault(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateAbsent, nil)
	cfg.ProviderModels = buildEffectiveProviderModels(nil)

	cfg.ProviderModels["openai"] = mergeProviderModelConfig(cfg.ProviderModels["openai"], ProviderModelConfig{
		DefaultModel: "gpt-custom",
	})
	cfg.DeleteProviderModelConfig("openai")

	want := DefaultConfig().ProviderModels["openai"].DefaultModel
	if got := cfg.ProviderModels["openai"].DefaultModel; got != want {
		t.Fatalf("DeleteProviderModelConfig(openai) fallback default = %q, want %q", got, want)
	}
	if got := cfg.providerModelSectionState(); got != providerModelSectionStateAbsent {
		t.Fatalf("providerModelSectionState() = %v, want %v", got, providerModelSectionStateAbsent)
	}
}

func TestDeleteProviderModelConfig_NoRawOverrideDeletesUnknownEffectiveEntry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateAbsent, nil)
	cfg.ProviderModels["custom-provider"] = ProviderModelConfig{DefaultModel: "custom-model"}

	cfg.DeleteProviderModelConfig("custom-provider")

	if _, ok := cfg.ProviderModels["custom-provider"]; ok {
		t.Fatalf("DeleteProviderModelConfig(custom-provider) should delete unknown effective entry, got %#v", cfg.ProviderModels["custom-provider"])
	}
}

func TestDeleteProviderModelKeys_DirectBranches(t *testing.T) {
	deleteProviderModelKeys(nil, []string{"openai"})

	raw := map[string]ProviderModelConfig{
		"openai":    {DefaultModel: "gpt-custom"},
		"anthropic": {DefaultModel: "claude-custom"},
	}
	deleteProviderModelKeys(raw, []string{"openai", "missing"})

	if _, ok := raw["openai"]; ok {
		t.Fatalf("deleteProviderModelKeys() should delete openai entry, got %#v", raw)
	}
	if got := raw["anthropic"].DefaultModel; got != "claude-custom" {
		t.Fatalf("deleteProviderModelKeys() should preserve unmatched entries, got anthropic=%q", got)
	}
}

func TestProviderModelDeleteActionFor_DirectBranches(t *testing.T) {
	if _, ok := providerModelDeleteActionFor(nil, ""); ok {
		t.Fatal("providerModelDeleteActionFor(empty provider) = ok, want false")
	}

	raw := map[string]ProviderModelConfig{
		"claude": {},
	}
	action, ok := providerModelDeleteActionFor(raw, "anthropic")
	if !ok {
		t.Fatal("providerModelDeleteActionFor(anthropic) should succeed")
	}
	if action.requestedKey != "anthropic" {
		t.Fatalf("requestedKey = %q, want %q", action.requestedKey, "anthropic")
	}
	if len(action.deleteKeys) != 1 || action.deleteKeys[0] != "claude" {
		t.Fatalf("deleteKeys = %v, want [claude]", action.deleteKeys)
	}
}
