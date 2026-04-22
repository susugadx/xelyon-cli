package config

import "testing"

func TestSyncProviderDefaultModel_UpdatesRequestedAnthropicEntryWithoutCollapsingSibling(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"claude": {
			DefaultModel:     "claude-old",
			AnthropicVersion: "2099-01-01",
			MaxOutputTokens:  1234,
		},
		"anthropic": {
			DefaultModel: "anthropic-old",
		},
	})

	if ok := cfg.SyncProviderDefaultModel("anthropic", "claude-new"); !ok {
		t.Fatal("SyncProviderDefaultModel(anthropic, claude-new) should succeed")
	}

	if got := cfg.GetSelectedModelForProvider("anthropic"); got != "claude-new" {
		t.Fatalf("GetSelectedModelForProvider(anthropic) = %q, want %q", got, "claude-new")
	}
	if got := cfg.GetSelectedModelForProvider("claude"); got != "claude-old" {
		t.Fatalf("GetSelectedModelForProvider(claude) = %q, want %q", got, "claude-old")
	}

	saved := cfg.ProviderModelsForSave()
	if len(saved) != 2 {
		t.Fatalf("len(ProviderModelsForSave()) = %d, want 2", len(saved))
	}

	pm, ok := saved["anthropic"]
	if !ok {
		t.Fatal("ProviderModelsForSave() should keep anthropic key after anthropic sync")
	}
	if pm.DefaultModel != "claude-new" {
		t.Fatalf("ProviderModelsForSave()[anthropic].DefaultModel = %q, want %q", pm.DefaultModel, "claude-new")
	}
	if pm.AnthropicVersion != "" {
		t.Fatalf("ProviderModelsForSave()[anthropic].AnthropicVersion = %q, want empty", pm.AnthropicVersion)
	}
	if pm.MaxOutputTokens != 0 {
		t.Fatalf("ProviderModelsForSave()[anthropic].MaxOutputTokens = %d, want %d", pm.MaxOutputTokens, 0)
	}
	if claude, ok := saved["claude"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want claude sibling preserved", saved)
	} else if claude.DefaultModel != "claude-old" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want %q", claude.DefaultModel, "claude-old")
	}
}

func TestSyncProviderDefaultModel_CreatesRequestedAnthropicEntryWhenDefaultProviderIsClaude(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "claude"
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"claude": {
			DefaultModel: "claude-old",
		},
	})

	if ok := cfg.SyncProviderDefaultModel("anthropic", "anthropic-new"); !ok {
		t.Fatal("SyncProviderDefaultModel(anthropic, anthropic-new) should succeed")
	}

	saved := cfg.ProviderModelsForSave()
	if anthropic, ok := saved["anthropic"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want anthropic entry created", saved)
	} else if anthropic.DefaultModel != "anthropic-new" {
		t.Fatalf("ProviderModelsForSave()[anthropic].DefaultModel = %q, want %q", anthropic.DefaultModel, "anthropic-new")
	}
	if claude, ok := saved["claude"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want claude entry preserved", saved)
	} else if claude.DefaultModel != "claude-old" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want %q", claude.DefaultModel, "claude-old")
	}
}

func TestClearProviderDefaultModelOverride_ClearsOnlyRequestedClaudeEntry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"claude": {
			DefaultModel:     "claude-old",
			AnthropicVersion: "2099-01-01",
		},
		"anthropic": {
			DefaultModel:    "anthropic-old",
			MaxOutputTokens: 1234,
		},
	})

	if ok := cfg.ClearProviderDefaultModelOverride("claude"); !ok {
		t.Fatal("ClearProviderDefaultModelOverride(claude) should succeed")
	}

	saved := cfg.ProviderModelsForSave()
	if len(saved) != 2 {
		t.Fatalf("len(ProviderModelsForSave()) = %d, want 2", len(saved))
	}

	pm, ok := saved["claude"]
	if !ok {
		t.Fatal("ProviderModelsForSave() should keep selected claude key")
	}
	if pm.DefaultModel != "" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want empty", pm.DefaultModel)
	}
	if pm.AnthropicVersion != "2099-01-01" {
		t.Fatalf("ProviderModelsForSave()[claude].AnthropicVersion = %q, want %q", pm.AnthropicVersion, "2099-01-01")
	}
	if pm.MaxOutputTokens != 0 {
		t.Fatalf("ProviderModelsForSave()[claude].MaxOutputTokens = %d, want %d", pm.MaxOutputTokens, 0)
	}
	if anthropic, ok := saved["anthropic"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want anthropic sibling preserved", saved)
	} else if anthropic.DefaultModel != "anthropic-old" {
		t.Fatalf("ProviderModelsForSave()[anthropic].DefaultModel = %q, want %q", anthropic.DefaultModel, "anthropic-old")
	}
}
