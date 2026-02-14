package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestHandleModelCommand_UpdatesProviderModel(t *testing.T) {
	// Setup temp home dir for config
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Initialize config
	cfg := config.DefaultConfig()
	cfg.ProviderModels["openai"] = config.ProviderModelConfig{
		DefaultModel: "gpt-original",
	}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	a := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-original",
		CurrentProvider: &MockProvider{name: "openai"},
	}

	// Execute /model command
	handleModelCommand(a, []string{"gpt-new"})

	// Verify Agent state
	if a.CurrentModel != "gpt-new" {
		t.Errorf("Agent.CurrentModel = %s, want gpt-new", a.CurrentModel)
	}

	// Verify Config state (ProviderModels should be updated)
	loadedCfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loadedCfg.DefaultModel != "gpt-new" {
		t.Errorf("Config.DefaultModel = %s, want gpt-new", loadedCfg.DefaultModel)
	}

	pm, ok := loadedCfg.ProviderModels["openai"]
	if !ok {
		t.Fatal("ProviderModels['openai'] missing")
	}
	if pm.DefaultModel != "gpt-new" {
		t.Errorf("ProviderModels['openai'].DefaultModel = %s, want gpt-new", pm.DefaultModel)
	}
}

func TestHandleConfigCommand_Model_UpdatesProviderModel(t *testing.T) {
	// Setup temp home dir for config
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Initialize config
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "openai"
	cfg.ProviderModels["openai"] = config.ProviderModelConfig{
		DefaultModel: "gpt-original",
	}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	a := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-original",
		CurrentProvider: &MockProvider{name: "openai"},
	}

	// Execute /config model command
	handleConfigCommand(a, []string{"model", "gpt-new"})

	// Verify Agent state (SyncWithGlobalConfig is called inside handleConfigCommand)
	if a.CurrentModel != "gpt-new" {
		t.Errorf("Agent.CurrentModel = %s, want gpt-new", a.CurrentModel)
	}

	// Verify Config state
	loadedCfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loadedCfg.DefaultModel != "gpt-new" {
		t.Errorf("Config.DefaultModel = %s, want gpt-new", loadedCfg.DefaultModel)
	}

	pm, ok := loadedCfg.ProviderModels["openai"]
	if !ok {
		t.Fatal("ProviderModels['openai'] missing")
	}
	if pm.DefaultModel != "gpt-new" {
		t.Errorf("ProviderModels['openai'].DefaultModel = %s, want gpt-new", pm.DefaultModel)
	}
}
