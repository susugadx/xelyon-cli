package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestHandleUseCommand_WithExplicitModel_UpdatesSessionModel(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := newProjectMapDisabledConfig()
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	var out bytes.Buffer
	agent := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-old",
		CurrentProvider: &mockCacheClearableProviderForModel{name: "openai"},
		Stats:           NewSessionStats("openai", "gpt-old"),
		History: []api.Message{
			{
				Role:    "assistant",
				Content: "tool call",
				ToolCalls: []api.OpenAIToolCall{
					{
						Index: 0,
						ID:    "tc1",
						Type:  "function",
						Function: api.OpenAIToolCallFunction{
							Name:      "read_file",
							Arguments: "{}",
						},
					},
				},
			},
			{
				Role:       "tool",
				Content:    "tool response",
				ToolCallID: "tc1",
				ToolName:   "read_file",
			},
		},
		Runtime: &AgentRuntime{
			Config: cfg,
			UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
		agentConversationState: agentConversationState{
			session: history.NewSession("gpt-old"),
		},
	}

	result := handleUseCommand(agent, []string{"ollama", "qwen2.5-coder:14b"})
	if !result {
		t.Fatal("handleUseCommand() = false, want true")
	}
	if agent.CurrentModel != "qwen2.5-coder:14b" {
		t.Fatalf("CurrentModel = %q, want %q", agent.CurrentModel, "qwen2.5-coder:14b")
	}
	if agent.session == nil || agent.session.Model != "qwen2.5-coder:14b" {
		t.Fatalf("session.Model = %q, want %q", agent.session.Model, "qwen2.5-coder:14b")
	}
	if agent.Stats == nil || agent.Stats.Model != "qwen2.5-coder:14b" {
		t.Fatalf("Stats.Model = %q, want %q", agent.Stats.Model, "qwen2.5-coder:14b")
	}
	if len(agent.History) != 2 {
		t.Fatalf("len(agent.History) = %d, want 2", len(agent.History))
	}
	if !strings.Contains(out.String(), "Context kept locally; provider remote continuation reset") {
		t.Fatalf("expected output to contain context kept notification, got %q", out.String())
	}
	if strings.Contains(out.String(), "History cleared after provider switch") {
		t.Fatalf("output should not contain history cleared notification, got %q", out.String())
	}
}

func TestHandleUseCommand_SwitchesAliasOwnerWithinSameRuntimeIdentity(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "claude"
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
		"claude":    {DefaultModel: "claude-custom"},
	})

	agent := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "claude",
		CurrentModel:      "claude-custom",
		CurrentProvider:   &mockCacheClearableProviderForModel{name: "claude"},
		Runtime: &AgentRuntime{
			Config: cfg,
			UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
		agentConversationState: agentConversationState{
			session: history.NewSession("claude-custom"),
		},
	}

	if result := handleUseCommand(agent, []string{"anthropic"}); !result {
		t.Fatal("handleUseCommand() = false, want true")
	}
	if agent.ProviderName != "claude" {
		t.Fatalf("ProviderName = %q, want %q", agent.ProviderName, "claude")
	}
	if agent.ProviderConfigKey != "anthropic" {
		t.Fatalf("ProviderConfigKey = %q, want %q", agent.ProviderConfigKey, "anthropic")
	}
	if providerConfigKeyFromProvider(agent.CurrentProvider) != "anthropic" {
		t.Fatalf("provider config key = %q, want %q", providerConfigKeyFromProvider(agent.CurrentProvider), "anthropic")
	}
	if agent.CurrentModel != "anthropic-custom" {
		t.Fatalf("CurrentModel = %q, want %q", agent.CurrentModel, "anthropic-custom")
	}
	if agent.session == nil || agent.session.Model != "anthropic-custom" {
		t.Fatalf("session.Model = %q, want %q", agent.session.Model, "anthropic-custom")
	}

	if result := handleUseCommand(agent, []string{"claude"}); !result {
		t.Fatal("handleUseCommand() = false, want true")
	}
	if agent.ProviderConfigKey != "claude" {
		t.Fatalf("ProviderConfigKey = %q, want %q", agent.ProviderConfigKey, "claude")
	}
	if providerConfigKeyFromProvider(agent.CurrentProvider) != "claude" {
		t.Fatalf("provider config key = %q, want %q", providerConfigKeyFromProvider(agent.CurrentProvider), "claude")
	}
	if agent.CurrentModel != "claude-custom" {
		t.Fatalf("CurrentModel = %q, want %q", agent.CurrentModel, "claude-custom")
	}
	if agent.session == nil || agent.session.Model != "claude-custom" {
		t.Fatalf("session.Model = %q, want %q", agent.session.Model, "claude-custom")
	}
}

func TestHandleUseCommand_AzureDisplayNameUsesCanonicalConfigOwner(t *testing.T) {
	t.Setenv("AZURE_OPENAI_API_KEY", "test-azure-key")
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com/openai/v1")

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-gpt55-deployment",
		CatalogModel: "gpt-5.5",
	})

	agent := &Agent{
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		CurrentModel:      "gpt-5.4",
		CurrentProvider:   &mockCacheClearableProviderForModel{name: "openai"},
		Runtime: &AgentRuntime{
			Config: cfg,
			UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
		agentConversationState: agentConversationState{
			session: history.NewSession("gpt-5.4"),
		},
	}

	if result := handleUseCommand(agent, []string{"Azure OpenAI"}); !result {
		t.Fatal("handleUseCommand() = false, want true")
	}
	if agent.ProviderName != "azure" {
		t.Fatalf("ProviderName = %q, want azure", agent.ProviderName)
	}
	if agent.ProviderConfigKey != "azure" {
		t.Fatalf("ProviderConfigKey = %q, want azure", agent.ProviderConfigKey)
	}
	if providerConfigKeyFromProvider(agent.CurrentProvider) != "azure" {
		t.Fatalf("provider config key = %q, want azure", providerConfigKeyFromProvider(agent.CurrentProvider))
	}
	if agent.CurrentModel != "corp-gpt55-deployment" {
		t.Fatalf("CurrentModel = %q, want configured Azure deployment", agent.CurrentModel)
	}
	if agent.session == nil || agent.session.ProviderName != "azure" || agent.session.ProviderConfigKey != "azure" {
		t.Fatalf("session provider identity = (%q, %q), want (azure, azure)", agent.session.ProviderName, agent.session.ProviderConfigKey)
	}
}

func TestHandleUseCommand_AzureWithoutDeploymentShowsActionableError(t *testing.T) {
	t.Setenv("AZURE_OPENAI_API_KEY", "test-azure-key")
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com/openai/v1")

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	agent := &Agent{
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		CurrentModel:      "gpt-5.4",
		CurrentProvider:   &mockCacheClearableProviderForModel{name: "openai"},
		Runtime: &AgentRuntime{
			Config: cfg,
			UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
		agentConversationState: agentConversationState{
			session: history.NewSession("gpt-5.4"),
		},
	}

	if result := handleUseCommand(agent, []string{"azure"}); !result {
		t.Fatal("handleUseCommand() = false, want true")
	}
	if !strings.Contains(out.String(), "deployment is not configured") {
		t.Fatalf("output = %q, want actionable Azure deployment error", out.String())
	}
	if agent.ProviderName != "openai" {
		t.Fatalf("ProviderName = %q, want openai on failed switch", agent.ProviderName)
	}
}

func TestHandleUseCommand_AzureWithExplicitDeploymentAllowsPlaceholderName(t *testing.T) {
	t.Setenv("AZURE_OPENAI_API_KEY", "test-azure-key")
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com/openai/v1")

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	agent := &Agent{
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		CurrentModel:      "gpt-5.4",
		CurrentProvider:   &mockCacheClearableProviderForModel{name: "openai"},
		Runtime: &AgentRuntime{
			Config: cfg,
			UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
		agentConversationState: agentConversationState{
			session: history.NewSession("gpt-5.4"),
		},
	}

	if result := handleUseCommand(agent, []string{"azure", "azure-gpt-5.4"}); !result {
		t.Fatal("handleUseCommand() = false, want true")
	}
	if agent.ProviderName != "azure" {
		t.Fatalf("ProviderName = %q, want azure", agent.ProviderName)
	}
	if agent.CurrentModel != "azure-gpt-5.4" {
		t.Fatalf("CurrentModel = %q, want explicit deployment name", agent.CurrentModel)
	}
}

func TestHandleUseCommand_HelpAndErrorBranches(t *testing.T) {
	t.Run("usage without args", func(t *testing.T) {
		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)
		if !handleUseCommand(agent, nil) {
			t.Fatal("handleUseCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Usage: /use <provider> [model]") {
			t.Fatalf("output = %q, want usage", out.String())
		}
	})

	t.Run("unknown provider is reported", func(t *testing.T) {
		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)
		if !handleUseCommand(agent, []string{"unknown-provider"}) {
			t.Fatal("handleUseCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Unknown provider") {
			t.Fatalf("output = %q, want unknown provider message", out.String())
		}
	})

	t.Run("already using provider without model prints hint", func(t *testing.T) {
		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)
		agent.ProviderName = "openai"
		agent.ProviderConfigKey = "openai"
		agent.CurrentModel = "gpt-current"
		if !handleUseCommand(agent, []string{"openai"}) {
			t.Fatal("handleUseCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Already using openai") || !strings.Contains(out.String(), "Hint: Use '/use <provider> <model>'") {
			t.Fatalf("output = %q, want already-using hint", out.String())
		}
	})

	t.Run("provider switch failure prints setup hint", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "")

		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)
		if !handleUseCommand(agent, []string{"openai"}) {
			t.Fatal("handleUseCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "OPENAI_API_KEY") {
			t.Fatalf("output = %q, want OPENAI_API_KEY setup hint", out.String())
		}
	})
}

func TestHandleProviderCommand_HelpAndAlreadyUsingHintUseProviderName(t *testing.T) {
	t.Run("usage without args", func(t *testing.T) {
		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)
		if !handleProviderCommand(agent, nil) {
			t.Fatal("handleProviderCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Usage: /provider <provider> [model]") {
			t.Fatalf("output = %q, want provider usage", out.String())
		}
	})

	t.Run("already using provider without model prints provider hint", func(t *testing.T) {
		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)
		agent.ProviderName = "openai"
		agent.ProviderConfigKey = "openai"
		agent.CurrentModel = "gpt-current"
		if !handleProviderCommand(agent, []string{"openai"}) {
			t.Fatal("handleProviderCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Already using openai") || !strings.Contains(out.String(), "Hint: Use '/provider <provider> <model>'") {
			t.Fatalf("output = %q, want provider already-using hint", out.String())
		}
	})
}

func TestHandleProviderAndUseCommands_ShareSwitchOutcome(t *testing.T) {
	t.Setenv("OLLAMA_BASE_URL", "http://localhost:11434")

	newAgent := func(out *bytes.Buffer) *Agent {
		cfg := newProjectMapDisabledConfig()
		return &Agent{
			ProviderName:      "openai",
			ProviderConfigKey: "openai",
			CurrentModel:      "gpt-old",
			CurrentProvider:   &mockCacheClearableProviderForModel{name: "openai"},
			Stats:             NewSessionStats("openai", "gpt-old"),
			Runtime: &AgentRuntime{
				Config: cfg,
				UI:     ui.NewRuntime(strings.NewReader(""), out, out),
			},
			agentConversationState: agentConversationState{
				session: history.NewSession("gpt-old"),
			},
		}
	}

	var providerOut bytes.Buffer
	providerAgent := newAgent(&providerOut)
	if !handleProviderCommand(providerAgent, []string{"ollama", "qwen2.5-coder:14b"}) {
		t.Fatal("handleProviderCommand() = false, want true")
	}

	var useOut bytes.Buffer
	useAgent := newAgent(&useOut)
	if !handleUseCommand(useAgent, []string{"ollama", "qwen2.5-coder:14b"}) {
		t.Fatal("handleUseCommand() = false, want true")
	}

	if providerAgent.ProviderName != useAgent.ProviderName ||
		providerAgent.ProviderConfigKey != useAgent.ProviderConfigKey ||
		providerAgent.CurrentModel != useAgent.CurrentModel ||
		providerAgent.session.Model != useAgent.session.Model {
		t.Fatalf("provider state = (%q,%q,%q,%q), use state = (%q,%q,%q,%q)",
			providerAgent.ProviderName, providerAgent.ProviderConfigKey, providerAgent.CurrentModel, providerAgent.session.Model,
			useAgent.ProviderName, useAgent.ProviderConfigKey, useAgent.CurrentModel, useAgent.session.Model)
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
