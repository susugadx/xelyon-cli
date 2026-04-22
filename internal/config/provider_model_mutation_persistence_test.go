package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateExistingProviderModelConfig_LoadedConfigCreatesProviderEntry(t *testing.T) {
	configPath := writeConfigYAMLForTest(t, `
default_provider: openai
default_model: gpt-old
`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if ok := cfg.UpdateExistingProviderModelConfig("openai", func(pm *ProviderModelConfig) {
		pm.DefaultModel = "gpt-new"
	}); !ok {
		t.Fatal("UpdateExistingProviderModelConfig(openai) should succeed")
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if got := loaded.GetModelForProvider("openai"); got != "gpt-new" {
		t.Fatalf("GetModelForProvider(openai) = %q, want %q", got, "gpt-new")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "provider_models:") || !strings.Contains(content, "openai:") {
		t.Fatalf("saved config should contain provider_models.openai entry, got:\n%s", content)
	}
}

func TestUpdateExistingProviderModelConfig_FirstRunPersistsProviderEntry(t *testing.T) {
	homeDir := setTestHome(t)

	cfg := DefaultConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "gpt-new"

	if ok := cfg.UpdateExistingProviderModelConfig("openai", func(pm *ProviderModelConfig) {
		pm.DefaultModel = "gpt-new"
	}); !ok {
		t.Fatal("UpdateExistingProviderModelConfig(openai) should succeed")
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if got := loaded.GetModelForProvider("openai"); got != "gpt-new" {
		t.Fatalf("GetModelForProvider(openai) = %q, want %q", got, "gpt-new")
	}

	data, err := os.ReadFile(filepath.Join(homeDir, ".xelyon", "config.yaml"))
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "provider_models:") || !strings.Contains(content, "openai:") {
		t.Fatalf("saved config should contain provider_models.openai entry, got:\n%s", content)
	}
}
