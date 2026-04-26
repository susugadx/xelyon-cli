package cmd

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestGetModel_PreservesAliasSpecificModelBeforeCanonicalFallback(t *testing.T) {
	tests := []struct {
		name            string
		providerFlagVal string
		envProvider     string
		defaultProvider string
		withAnthropic   bool
		wantModel       string
	}{
		{
			name:            "anthropic alias prefers anthropic-specific model",
			providerFlagVal: "Anthropic",
			defaultProvider: "deepseek",
			withAnthropic:   true,
			wantModel:       "anthropic-custom",
		},
		{
			name:            "env provider is normalized",
			envProvider:     " OpenAI ",
			defaultProvider: "deepseek",
			wantModel:       "gpt-custom",
		},
		{
			name:            "config provider is normalized",
			defaultProvider: " Gemini ",
			wantModel:       "gemini-custom",
		},
		{
			name:            "anthropic alias falls back to claude model",
			providerFlagVal: "anthropic",
			defaultProvider: "deepseek",
			wantModel:       "claude-custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetRootFlagsForTest()
			t.Cleanup(resetRootFlagsForTest)

			cfg := config.DefaultConfig()
			cfg.DefaultModel = "fallback-model"
			cfg.DefaultProvider = tt.defaultProvider
			if tt.withAnthropic {
				cfg.SetProviderModelConfig("anthropic", config.ProviderModelConfig{DefaultModel: "anthropic-custom"})
			}
			cfg.ProviderModels["claude"] = config.ProviderModelConfig{DefaultModel: "claude-custom"}
			cfg.ProviderModels["openai"] = config.ProviderModelConfig{DefaultModel: "gpt-custom"}
			cfg.ProviderModels["gemini"] = config.ProviderModelConfig{DefaultModel: "gemini-custom"}

			providerFlag = tt.providerFlagVal
			t.Setenv("XELYON_PROVIDER", tt.envProvider)

			if got := getModel(cfg); got != tt.wantModel {
				t.Fatalf("getModel() = %q, want %q", got, tt.wantModel)
			}
		})
	}
}

func TestGetModel_DefaultProviderKeepsGlobalDefaultModelWhenExplicitEntryHasOnlyNonModelSettings(t *testing.T) {
	resetRootFlagsForTest()
	t.Cleanup(resetRootFlagsForTest)

	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "gpt-custom"
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		MaxOutputTokens: 99999,
	})

	if got := getModel(cfg); got != "gpt-custom" {
		t.Fatalf("getModel() = %q, want %q", got, "gpt-custom")
	}
}

func TestGetModel_EnvironmentModelOverridesProviderResolution(t *testing.T) {
	resetRootFlagsForTest()
	t.Cleanup(resetRootFlagsForTest)

	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "openai"
	cfg.ProviderModels["openai"] = config.ProviderModelConfig{DefaultModel: "gpt-custom"}
	t.Setenv("XELYON_MODEL", " env-model ")

	if got := getModel(cfg); got != "env-model" {
		t.Fatalf("getModel() = %q, want %q", got, "env-model")
	}
}

func TestGetModel_FlagOverridesEnvironmentModel(t *testing.T) {
	resetRootFlagsForTest()
	t.Cleanup(resetRootFlagsForTest)

	modelFlag = "flag-model"
	t.Setenv("XELYON_MODEL", "env-model")

	if got := getModel(config.DefaultConfig()); got != "flag-model" {
		t.Fatalf("getModel() = %q, want %q", got, "flag-model")
	}
}
