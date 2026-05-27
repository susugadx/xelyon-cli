package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestGetDefaultModelWithContext_UsesInjectedConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ProviderModels["openai"] = config.ProviderModelConfig{DefaultModel: "gpt-5.2-runtime"}
	ctx := config.WithContext(context.Background(), cfg)

	got := GetDefaultModelWithContext(ctx, "", "openai", "fallback")
	if got != "gpt-5.2-runtime" {
		t.Fatalf("GetDefaultModelWithContext() = %q, want %q", got, "gpt-5.2-runtime")
	}
}

func TestGetDefaultModelWithContext_UsesAnthropicAliasConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("anthropic", config.ProviderModelConfig{DefaultModel: "anthropic-runtime"})
	ctx := config.WithContext(context.Background(), cfg)

	got := GetDefaultModelWithContext(ctx, "", "claude", "fallback")
	if got != "anthropic-runtime" {
		t.Fatalf("GetDefaultModelWithContext() = %q, want %q", got, "anthropic-runtime")
	}
}

func TestGetDefaultModelWithContext_UsesEffectiveProviderDefaultForOtherProvider(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: deepseek
default_model: global-custom-model
provider_models:
  deepseek:
    default_model: deepseek-custom
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	ctx := config.WithContext(context.Background(), cfg)

	got := GetDefaultModelWithContext(ctx, "", "openai", "fallback")
	want := config.DefaultConfig().ProviderModels["openai"].DefaultModel
	if got != want {
		t.Fatalf("GetDefaultModelWithContext() = %q, want %q", got, want)
	}
}

func TestResolveProviderRequestModel_UsesDescriptorDefaultWhenConfigMissingProviderModel(t *testing.T) {
	ctx := config.WithContext(context.Background(), &config.Config{})

	got := ResolveProviderRequestModel(ctx, "", "groq")
	if got != "meta-llama/llama-4-scout-17b-16e-instruct" {
		t.Fatalf("ResolveProviderRequestModel() = %q, want Groq descriptor default", got)
	}
}

func TestGetDefaultModelWithContext_FallbackOnlyForUnknownProvider(t *testing.T) {
	ctx := config.WithContext(context.Background(), &config.Config{})

	got := GetDefaultModelWithContext(ctx, "", "unknown-provider", "fallback")
	if got != "fallback" {
		t.Fatalf("GetDefaultModelWithContext(unknown) = %q, want fallback", got)
	}
}
