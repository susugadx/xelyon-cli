package config

import "testing"

func TestProviderDefaultModelSyncPlanFor_DirectBranches(t *testing.T) {
	t.Run("invalid provider returns invalid plan", func(t *testing.T) {
		plan := providerDefaultModelSyncPlanFor("   ", "gpt-custom")
		if plan.valid {
			t.Fatalf("providerDefaultModelSyncPlanFor(invalid provider) = %#v, want invalid", plan)
		}
	})

	t.Run("empty model returns invalid plan", func(t *testing.T) {
		plan := providerDefaultModelSyncPlanFor("openai", "")
		if plan.valid {
			t.Fatalf("providerDefaultModelSyncPlanFor(empty model) = %#v, want invalid", plan)
		}
	})

	t.Run("provider default model requests exact clear", func(t *testing.T) {
		defaultModel := DefaultConfig().ProviderModels["openai"].DefaultModel
		plan := providerDefaultModelSyncPlanFor(" openai ", defaultModel)
		if !plan.valid {
			t.Fatalf("providerDefaultModelSyncPlanFor(default model) = %#v, want valid", plan)
		}
		if plan.key != "openai" {
			t.Fatalf("plan.key = %q, want %q", plan.key, "openai")
		}
		if !plan.clearExact {
			t.Fatalf("plan.clearExact = %v, want true", plan.clearExact)
		}
	})

	t.Run("non-default model requests upsert", func(t *testing.T) {
		plan := providerDefaultModelSyncPlanFor("anthropic", "claude-custom")
		if !plan.valid {
			t.Fatalf("providerDefaultModelSyncPlanFor(custom model) = %#v, want valid", plan)
		}
		if plan.key != "anthropic" {
			t.Fatalf("plan.key = %q, want %q", plan.key, "anthropic")
		}
		if plan.model != "claude-custom" {
			t.Fatalf("plan.model = %q, want %q", plan.model, "claude-custom")
		}
		if plan.clearExact {
			t.Fatalf("plan.clearExact = %v, want false", plan.clearExact)
		}
	})
}

func TestProviderDefaultModelRawHelpers_DirectBranches(t *testing.T) {
	t.Run("set preserves sibling fields", func(t *testing.T) {
		raw := map[string]ProviderModelConfig{
			"openai": {MaxOutputTokens: 999},
		}
		setProviderDefaultModelInRaw(raw, "openai", "gpt-custom")
		pm := raw["openai"]
		if pm.DefaultModel != "gpt-custom" {
			t.Fatalf("setProviderDefaultModelInRaw().DefaultModel = %q, want %q", pm.DefaultModel, "gpt-custom")
		}
		if pm.MaxOutputTokens != 999 {
			t.Fatalf("setProviderDefaultModelInRaw().MaxOutputTokens = %d, want %d", pm.MaxOutputTokens, 999)
		}
	})

	t.Run("clear preserves non-default fields", func(t *testing.T) {
		raw := map[string]ProviderModelConfig{
			"openai": {
				DefaultModel:    "gpt-custom",
				MaxOutputTokens: 999,
			},
		}
		clearProviderDefaultModelInRaw(raw, "openai")
		pm, ok := raw["openai"]
		if !ok {
			t.Fatalf("clearProviderDefaultModelInRaw() = %#v, want openai entry kept", raw)
		}
		if pm.DefaultModel != "" {
			t.Fatalf("clearProviderDefaultModelInRaw().DefaultModel = %q, want empty", pm.DefaultModel)
		}
		if pm.MaxOutputTokens != 999 {
			t.Fatalf("clearProviderDefaultModelInRaw().MaxOutputTokens = %d, want %d", pm.MaxOutputTokens, 999)
		}
	})

	t.Run("clear removes zero entry", func(t *testing.T) {
		raw := map[string]ProviderModelConfig{
			"openai": {DefaultModel: "gpt-custom"},
		}
		clearProviderDefaultModelInRaw(raw, "openai")
		if _, ok := raw["openai"]; ok {
			t.Fatalf("clearProviderDefaultModelInRaw() = %#v, want openai removed", raw)
		}
	})
}
