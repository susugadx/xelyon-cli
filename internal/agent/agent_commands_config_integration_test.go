package agent

import (
	"bytes"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestHandleModelCommand_UpdatesProviderModel(t *testing.T) {
	// Setup temp home dir for config
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Initialize config
	cfg := newProjectMapDisabledConfig()
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		DefaultModel: "gpt-original",
	})
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	a := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-original",
		CurrentProvider: &MockProvider{name: "openai"},
		Runtime:         NewAgentRuntimeWithConfig(cfg),
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
	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "openai"
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		DefaultModel: "gpt-original",
	})
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	a := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-original",
		CurrentProvider: &MockProvider{name: "openai"},
		Runtime:         NewAgentRuntimeWithConfig(cfg),
	}

	// Execute /config model command
	handleConfigCommand(a, []string{"model", "gpt-new"})

	// Verify Agent state (SyncWithRuntimeConfig is called inside handleConfigCommand)
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

func TestHandleModelCommand_PrefersAnthropicAliasEntryWhenDefaultProviderUsesAlias(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-old"},
		"claude":    {DefaultModel: "claude-old"},
	})
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	a := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "anthropic",
		CurrentModel:      "anthropic-old",
		CurrentProvider:   &MockProvider{name: "claude"},
		Runtime:           NewAgentRuntimeWithConfig(cfg),
	}

	handleModelCommand(a, []string{"claude-new"})

	loadedCfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	pmAnthropic, ok := loadedCfg.ProviderModelsForSave()["anthropic"]
	if !ok {
		t.Fatal("ProviderModelsForSave()['anthropic'] missing")
	}
	if pmAnthropic.DefaultModel != "claude-new" {
		t.Fatalf("ProviderModelsForSave()['anthropic'].DefaultModel = %q, want %q", pmAnthropic.DefaultModel, "claude-new")
	}
	if pmClaude, ok := loadedCfg.ProviderModelsForSave()["claude"]; !ok {
		t.Fatal("ProviderModelsForSave()['claude'] should remain when editing anthropic alias entry")
	} else if pmClaude.DefaultModel != "claude-old" {
		t.Fatalf("ProviderModelsForSave()['claude'].DefaultModel = %q, want %q", pmClaude.DefaultModel, "claude-old")
	}
}

func TestHandleModelCommand_PreservesAnthropicAliasEntryWhenDefaultProviderDiffers(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-old"},
		"claude":    {DefaultModel: "claude-old"},
	})
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	a := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "anthropic",
		CurrentModel:      "anthropic-old",
		CurrentProvider:   &MockProvider{name: "claude"},
		Runtime:           NewAgentRuntimeWithConfig(cfg),
	}

	handleModelCommand(a, []string{"claude-new"})

	loadedCfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	pmAnthropic, ok := loadedCfg.ProviderModelsForSave()["anthropic"]
	if !ok {
		t.Fatal("ProviderModelsForSave()['anthropic'] missing")
	}
	if pmAnthropic.DefaultModel != "claude-new" {
		t.Fatalf("ProviderModelsForSave()['anthropic'].DefaultModel = %q, want %q", pmAnthropic.DefaultModel, "claude-new")
	}
	if pmClaude, ok := loadedCfg.ProviderModelsForSave()["claude"]; !ok {
		t.Fatal("ProviderModelsForSave()['claude'] should remain when editing anthropic alias entry")
	} else if pmClaude.DefaultModel != "claude-old" {
		t.Fatalf("ProviderModelsForSave()['claude'].DefaultModel = %q, want %q", pmClaude.DefaultModel, "claude-old")
	}
}

func TestHandleUseThenModelCommand_PersistsAnthropicAliasOwnerModel(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "claude"
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-old"},
		"claude":    {DefaultModel: "claude-old"},
	})
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	var out bytes.Buffer
	a := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "claude",
		CurrentModel:      "claude-old",
		CurrentProvider:   &MockProvider{name: "claude"},
		Runtime: &AgentRuntime{
			Config: cfg,
			UI:     ui.NewRuntime(nil, &out, &out),
		},
		agentConversationState: agentConversationState{
			session: history.NewSession("claude-old"),
		},
	}

	if !handleUseCommand(a, []string{"anthropic"}) {
		t.Fatal("handleUseCommand() = false, want true")
	}
	if a.ProviderConfigKey != "anthropic" {
		t.Fatalf("ProviderConfigKey = %q, want %q after /use", a.ProviderConfigKey, "anthropic")
	}

	handleModelCommand(a, []string{"anthropic-new"})

	if a.CurrentModel != "anthropic-new" {
		t.Fatalf("Agent.CurrentModel = %q, want %q", a.CurrentModel, "anthropic-new")
	}
	if a.session == nil || a.session.Model != "anthropic-new" {
		t.Fatalf("session.Model = %q, want %q", a.session.Model, "anthropic-new")
	}

	loadedCfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	pmAnthropic, ok := loadedCfg.ProviderModelsForSave()["anthropic"]
	if !ok {
		t.Fatal("ProviderModelsForSave()['anthropic'] missing")
	}
	if pmAnthropic.DefaultModel != "anthropic-new" {
		t.Fatalf("ProviderModelsForSave()['anthropic'].DefaultModel = %q, want %q", pmAnthropic.DefaultModel, "anthropic-new")
	}

	pmClaude, ok := loadedCfg.ProviderModelsForSave()["claude"]
	if !ok {
		t.Fatal("ProviderModelsForSave()['claude'] missing")
	}
	if pmClaude.DefaultModel != "claude-old" {
		t.Fatalf("ProviderModelsForSave()['claude'].DefaultModel = %q, want %q", pmClaude.DefaultModel, "claude-old")
	}
}
