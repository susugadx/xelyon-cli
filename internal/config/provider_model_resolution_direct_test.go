package config

import "testing"

func TestGetSelectedModelForProvider_DirectBranches(t *testing.T) {
	t.Run("nil config returns empty", func(t *testing.T) {
		var cfg *Config
		if got := cfg.GetSelectedModelForProvider("openai"); got != "" {
			t.Fatalf("nil GetSelectedModelForProvider(openai) = %q, want empty", got)
		}
	})

	t.Run("unknown provider returns empty", func(t *testing.T) {
		cfg := DefaultConfig()
		if got := cfg.GetSelectedModelForProvider("unknown-provider"); got != "" {
			t.Fatalf("GetSelectedModelForProvider(unknown-provider) = %q, want empty", got)
		}
	})

	t.Run("explicit entry without default falls back to applicable global default", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.DefaultProvider = "anthropic"
		cfg.DefaultModel = "claude-global"
		cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
			"anthropic": {MaxOutputTokens: 999},
		})
		cfg.refreshEffectiveProviderModels()

		if got := cfg.GetSelectedModelForProvider("anthropic"); got != "claude-global" {
			t.Fatalf("GetSelectedModelForProvider(anthropic) = %q, want %q", got, "claude-global")
		}
	})

	t.Run("explicit entry without default falls back to provider default when global default belongs elsewhere", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.DefaultProvider = "anthropic"
		cfg.DefaultModel = "gpt-5.4"
		cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
			"anthropic": {MaxOutputTokens: 999},
		})
		cfg.refreshEffectiveProviderModels()

		want := DefaultConfig().ProviderModels["claude"].DefaultModel
		if got := cfg.GetSelectedModelForProvider("anthropic"); got != want {
			t.Fatalf("GetSelectedModelForProvider(anthropic) = %q, want %q", got, want)
		}
	})
}

func TestSelectedModelOwnerWithinRuntimeIdentity_DirectBranches(t *testing.T) {
	t.Run("nil config and blank model return empty", func(t *testing.T) {
		var cfg *Config
		if got := cfg.selectedModelOwnerWithinRuntimeIdentity("claude", "shared-custom"); got != "" {
			t.Fatalf("nil selectedModelOwnerWithinRuntimeIdentity() = %q, want empty", got)
		}

		cfg = DefaultConfig()
		if got := cfg.selectedModelOwnerWithinRuntimeIdentity("claude", "   "); got != "" {
			t.Fatalf("selectedModelOwnerWithinRuntimeIdentity(blank) = %q, want empty", got)
		}
	})

	t.Run("returns exact requested alias when both siblings select same model", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
			"anthropic": {DefaultModel: "shared-custom"},
			"claude":    {DefaultModel: "shared-custom"},
		})
		cfg.refreshEffectiveProviderModels()

		if got := cfg.selectedModelOwnerWithinRuntimeIdentity("anthropic", "shared-custom"); got != "anthropic" {
			t.Fatalf("selectedModelOwnerWithinRuntimeIdentity(anthropic) = %q, want %q", got, "anthropic")
		}
		if got := cfg.selectedModelOwnerWithinRuntimeIdentity("claude", "shared-custom"); got != "claude" {
			t.Fatalf("selectedModelOwnerWithinRuntimeIdentity(claude) = %q, want %q", got, "claude")
		}
	})

	t.Run("returns empty when sibling runtime identity does not own model", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
			"anthropic": {DefaultModel: "anthropic-custom"},
			"claude":    {DefaultModel: "claude-custom"},
		})
		cfg.refreshEffectiveProviderModels()

		if got := cfg.selectedModelOwnerWithinRuntimeIdentity("claude", "missing-model"); got != "" {
			t.Fatalf("selectedModelOwnerWithinRuntimeIdentity(missing) = %q, want empty", got)
		}
	})
}

func TestConfiguredDefaultModelAppliesToProvider_DirectBranches(t *testing.T) {
	t.Run("empty model returns false", func(t *testing.T) {
		cfg := DefaultConfig()
		if cfg.configuredDefaultModelAppliesToProvider("openai", "") {
			t.Fatal("configuredDefaultModelAppliesToProvider(empty) = true, want false")
		}
	})

	t.Run("nil config falls back to inference and unknown model is treated as applicable", func(t *testing.T) {
		var cfg *Config
		if !cfg.configuredDefaultModelAppliesToProvider("openai", "corp-custom-opaque") {
			t.Fatal("nil configuredDefaultModelAppliesToProvider(unknown model) = false, want true")
		}
		if cfg.configuredDefaultModelAppliesToProvider("openai", "claude-sonnet-4-6") {
			t.Fatal("nil configuredDefaultModelAppliesToProvider(foreign inferred model) = true, want false")
		}
	})
}
