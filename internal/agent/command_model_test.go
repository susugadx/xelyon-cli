package agent

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestHandleModelCommand_ClearCache(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := newProjectMapDisabledConfig()
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	agent := &Agent{
		ProviderName: "mock",
		CurrentModel: "old-model",
		Stats:        NewSessionStats("mock", "old-model"),
		Runtime:      NewAgentRuntimeWithConfig(cfg),
		agentConversationState: agentConversationState{
			session: history.NewSession("old-model"),
		},
	}

	mockProvider := &mockCacheClearableProviderForModel{}
	agent.CurrentProvider = mockProvider

	result := handleModelCommand(agent, []string{"new-model"})

	assert.True(t, result)
	assert.Equal(t, "new-model", agent.CurrentModel)
	assert.Equal(t, "new-model", agent.session.Model)
	assert.Equal(t, "new-model", agent.Stats.Model)
	assert.True(t, mockProvider.cleared, "ClearCache should be called when switching model")
}

func TestHandleModelCommand_RebuildsClaudePromptForOpus(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := newProjectMapDisabledConfig()
	cfg.PromptCache.Enabled = true
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}
	runtime := NewAgentRuntimeWithConfig(cfg)

	agent := &Agent{
		ProviderName:    "claude",
		CurrentModel:    "claude-sonnet-4-6",
		CurrentProvider: &mockCacheClearableProviderForModel{name: "claude"},
		SystemPrompt:    prompt.BuildProviderSystemPromptWithConfig(prompt.SystemPrompt, "claude", "claude-sonnet-4-6", cfg),
		Runtime:         runtime,
	}

	result := handleModelCommand(agent, []string{"claude-opus-4-6"})

	assert.True(t, result)
	assert.Contains(t, agent.SystemPrompt, "## Workflow Rules")
}

func TestHandleModelCommand_NoArgs_UsesRuntimeOutput(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		ProviderName:    "mock",
		CurrentModel:    "test-model",
		CurrentProvider: &mockCacheClearableProviderForModel{name: "mock"},
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	result := handleModelCommand(agent, nil)
	if !result {
		t.Fatal("handleModelCommand() = false, want true")
	}

	output := out.String()
	if !strings.Contains(output, "Current model: test-model") {
		t.Fatalf("expected runtime output to contain current model, got %q", output)
	}
	if !strings.Contains(output, "Usage: /model <model-name>") {
		t.Fatalf("expected runtime output to contain usage, got %q", output)
	}
}

func TestHandleModelCommand_ListsInstalledModelsAndWarnings(t *testing.T) {
	t.Run("lists installed models", func(t *testing.T) {
		var out bytes.Buffer
		agent := &Agent{
			ProviderName:    "mock",
			CurrentModel:    "test-model",
			CurrentProvider: &mockModelListerProvider{models: []string{"model-a", "model-b"}},
			Runtime: &AgentRuntime{
				UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
			},
		}

		if !handleModelCommand(agent, nil) {
			t.Fatal("handleModelCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Installed models:") || !strings.Contains(out.String(), "model-a") {
			t.Fatalf("output = %q, want installed model list", out.String())
		}
	})

	t.Run("warns when model list fails", func(t *testing.T) {
		var out bytes.Buffer
		agent := &Agent{
			ProviderName:    "mock",
			CurrentModel:    "test-model",
			CurrentProvider: &mockModelListerProvider{err: errors.New("list failed")},
			Runtime: &AgentRuntime{
				UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
			},
		}

		if !handleModelCommand(agent, nil) {
			t.Fatal("handleModelCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Could not list models") {
			t.Fatalf("output = %q, want model listing warning", out.String())
		}
	})
}

func TestHandleModelCommand_ConfigLoadAndSaveWarnings(t *testing.T) {
	t.Run("load config failure keeps session change", func(t *testing.T) {
		withConfigCommandHooks(t)
		loadConfigForCommand = func() (*config.Config, error) {
			return nil, errors.New("load failed")
		}

		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)

		if !handleModelCommand(agent, []string{"new-model"}) {
			t.Fatal("handleModelCommand() = false, want true")
		}
		if agent.CurrentModel != "new-model" {
			t.Fatalf("CurrentModel = %q, want %q", agent.CurrentModel, "new-model")
		}
		if !strings.Contains(out.String(), "Warning: Failed to load config") {
			t.Fatalf("output = %q, want load warning", out.String())
		}
	})

	t.Run("save config failure keeps session only change", func(t *testing.T) {
		withConfigCommandHooks(t)
		loadConfigForCommand = func() (*config.Config, error) {
			return newProjectMapDisabledConfig(), nil
		}
		saveConfigForCommand = func(cfg *config.Config) error {
			return errors.New("save failed")
		}

		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)

		if !handleModelCommand(agent, []string{"new-model"}) {
			t.Fatal("handleModelCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Model switched for this session only") {
			t.Fatalf("output = %q, want session-only warning", out.String())
		}
	})
}

func TestHandleModelCommand_UpdatesProviderModel(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

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

	handleModelCommand(a, []string{"gpt-new"})

	if a.CurrentModel != "gpt-new" {
		t.Errorf("Agent.CurrentModel = %s, want gpt-new", a.CurrentModel)
	}

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
