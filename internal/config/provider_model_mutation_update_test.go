package config

import "testing"

func TestSetProviderModelConfig_AnthropicReusesExistingClaudeKey(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"claude": {
			DefaultModel:     "claude-custom",
			AnthropicVersion: "2099-01-01",
		},
	})

	cfg.SetProviderModelConfig("anthropic", ProviderModelConfig{
		DefaultModel:     "anthropic-updated",
		MaxOutputTokens:  4321,
		AnthropicVersion: "2100-01-01",
	})

	saved := cfg.ProviderModelsForSave()
	if len(saved) != 1 {
		t.Fatalf("len(ProviderModelsForSave()) = %d, want 1", len(saved))
	}
	if _, ok := saved["anthropic"]; ok {
		t.Fatal("ProviderModelsForSave() should not keep duplicate anthropic entry when claude already existed")
	}

	pm, ok := saved["claude"]
	if !ok {
		t.Fatal("ProviderModelsForSave() should reuse existing claude key")
	}
	if pm.DefaultModel != "anthropic-updated" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want %q", pm.DefaultModel, "anthropic-updated")
	}
	if pm.MaxOutputTokens != 4321 {
		t.Fatalf("ProviderModelsForSave()[claude].MaxOutputTokens = %d, want %d", pm.MaxOutputTokens, 4321)
	}
	if pm.AnthropicVersion != "2100-01-01" {
		t.Fatalf("ProviderModelsForSave()[claude].AnthropicVersion = %q, want %q", pm.AnthropicVersion, "2100-01-01")
	}
}

func TestSetProviderModelConfig_PreservesClaudeSiblingWhenUpdatingAnthropicEntry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"claude": {
			DefaultModel:     "claude-custom",
			AnthropicVersion: "2099-01-01",
		},
		"anthropic": {
			MaxOutputTokens: 4321,
		},
	})
	cfg.refreshEffectiveProviderModels()

	cfg.SetProviderModelConfig("anthropic", ProviderModelConfig{
		MaxOutputTokens: 9999,
	})

	saved := cfg.ProviderModelsForSave()
	if len(saved) != 2 {
		t.Fatalf("len(ProviderModelsForSave()) = %d, want 2", len(saved))
	}

	pm, ok := saved["anthropic"]
	if !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want anthropic entry to be updated in place", saved)
	}
	if pm.DefaultModel != "" {
		t.Fatalf("ProviderModelsForSave()[anthropic].DefaultModel = %q, want empty", pm.DefaultModel)
	}
	if pm.MaxOutputTokens != 9999 {
		t.Fatalf("ProviderModelsForSave()[anthropic].MaxOutputTokens = %d, want %d", pm.MaxOutputTokens, 9999)
	}

	claude, ok := saved["claude"]
	if !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want claude sibling preserved", saved)
	}
	if claude.DefaultModel != "claude-custom" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want %q", claude.DefaultModel, "claude-custom")
	}
	if claude.AnthropicVersion != "2099-01-01" {
		t.Fatalf("ProviderModelsForSave()[claude].AnthropicVersion = %q, want %q", claude.AnthropicVersion, "2099-01-01")
	}
}

func TestPatchProviderModelConfig_PreservesClaudeSiblingWhenUpdatingAnthropicEntry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"claude": {
			DefaultModel:     "claude-custom",
			AnthropicVersion: "2099-01-01",
		},
		"anthropic": {
			DefaultModel:    "anthropic-custom",
			MaxOutputTokens: 4321,
		},
	})
	cfg.refreshEffectiveProviderModels()

	if ok := cfg.PatchProviderModelConfig("anthropic", func(pm *ProviderModelConfig) {
		pm.MaxOutputTokens = 9999
	}); !ok {
		t.Fatal("PatchProviderModelConfig(anthropic) should succeed")
	}

	saved := cfg.ProviderModelsForSave()
	if len(saved) != 2 {
		t.Fatalf("len(ProviderModelsForSave()) = %d, want 2", len(saved))
	}
	if got := cfg.GetSelectedModelForProvider("anthropic"); got != "anthropic-custom" {
		t.Fatalf("GetSelectedModelForProvider(anthropic) = %q, want %q", got, "anthropic-custom")
	}
	if anthropic, ok := saved["anthropic"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want anthropic entry preserved", saved)
	} else {
		if anthropic.DefaultModel != "anthropic-custom" {
			t.Fatalf("ProviderModelsForSave()[anthropic].DefaultModel = %q, want %q", anthropic.DefaultModel, "anthropic-custom")
		}
		if anthropic.MaxOutputTokens != 9999 {
			t.Fatalf("ProviderModelsForSave()[anthropic].MaxOutputTokens = %d, want %d", anthropic.MaxOutputTokens, 9999)
		}
	}
	if claude, ok := saved["claude"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want claude sibling preserved", saved)
	} else if claude.DefaultModel != "claude-custom" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want %q", claude.DefaultModel, "claude-custom")
	}
}
