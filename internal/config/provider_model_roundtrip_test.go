package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveConfig_AnthropicAliasRoundTripUsesAliasWhenCanonicalNotRaw(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.SetProviderModelConfig("anthropic", ProviderModelConfig{
		DefaultModel:     "anthropic-custom",
		AnthropicVersion: "2099-01-01",
		AnthropicBeta:    []string{"alias-beta"},
	})

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	pm, ok := loaded.GetProviderModelConfig("claude")
	if !ok {
		t.Fatal("GetProviderModelConfig(claude) should succeed")
	}
	if pm.DefaultModel != "anthropic-custom" {
		t.Fatalf("DefaultModel = %q, want %q", pm.DefaultModel, "anthropic-custom")
	}
	if pm.AnthropicVersion != "2099-01-01" {
		t.Fatalf("AnthropicVersion = %q, want %q", pm.AnthropicVersion, "2099-01-01")
	}
	if len(pm.AnthropicBeta) != 1 || pm.AnthropicBeta[0] != "alias-beta" {
		t.Fatalf("AnthropicBeta = %v, want [alias-beta]", pm.AnthropicBeta)
	}

	key, ok := loaded.ProviderModelWriteKey("claude")
	if !ok {
		t.Fatal("ProviderModelWriteKey(claude) should succeed")
	}
	if key != "anthropic" {
		t.Fatalf("ProviderModelWriteKey(claude) = %q, want %q", key, "anthropic")
	}
}

func TestSaveConfig_DoesNotPersistEffectiveDefaultClaudeWhenSavingAnthropicAlias(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.SetProviderModelConfig("anthropic", ProviderModelConfig{DefaultModel: "anthropic-custom"})

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	configPath := filepath.Join(tmpDir, ".xelyon", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "anthropic:") {
		t.Fatalf("saved config should contain anthropic provider_models entry, got:\n%s", content)
	}
	if strings.Contains(content, "\n  claude:\n") {
		t.Fatalf("saved config should not persist effective default claude entry, got:\n%s", content)
	}
}

func TestSaveConfig_PreservesExplicitEmptyProviderModels(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: openai
default_model: global-custom-model
provider_models: {}
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	openAIWant := DefaultConfig().ProviderModels["openai"].DefaultModel
	if got := cfg.GetModelForProvider("openai"); got != openAIWant {
		t.Fatalf("GetModelForProvider(openai) = %q, want provider default %q", got, openAIWant)
	}
	if got := cfg.ResolveModelForProvider("openai"); got != "global-custom-model" {
		t.Fatalf("ResolveModelForProvider(openai) = %q, want %q", got, "global-custom-model")
	}
	deepseekWant := DefaultConfig().ProviderModels["deepseek"].DefaultModel
	if got := cfg.ResolveModelForProvider("deepseek"); got != deepseekWant {
		t.Fatalf("ResolveModelForProvider(deepseek) = %q, want provider default %q", got, deepseekWant)
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "provider_models: {}") {
		t.Fatalf("saved config should preserve explicit empty provider_models map, got:\n%s", content)
	}

	reloaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got := reloaded.GetModelForProvider("openai"); got != openAIWant {
		t.Fatalf("reloaded GetModelForProvider(openai) = %q, want provider default %q", got, openAIWant)
	}
	if got := reloaded.ResolveModelForProvider("openai"); got != "global-custom-model" {
		t.Fatalf("reloaded ResolveModelForProvider(openai) = %q, want %q", got, "global-custom-model")
	}
	if got := reloaded.ResolveModelForProvider("deepseek"); got != deepseekWant {
		t.Fatalf("reloaded ResolveModelForProvider(deepseek) = %q, want provider default %q", got, deepseekWant)
	}
}
