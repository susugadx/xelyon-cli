package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/susugadx/xelyon-cli/internal/api"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/azure"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/ollama"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// mockCacheClearableProviderForModel はキャッシュクリアと Provider を実装したモック（モデルコマンド用）
type mockCacheClearableProviderForModel struct {
	cleared bool
	name    string
}

func (m *mockCacheClearableProviderForModel) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock-cache"
}

func (m *mockCacheClearableProviderForModel) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	return "", nil
}

func (m *mockCacheClearableProviderForModel) SupportsImages() bool {
	return false
}

func (m *mockCacheClearableProviderForModel) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return "", nil
}

func (m *mockCacheClearableProviderForModel) IsFunctionCallingEnabled() bool {
	return false
}

func (m *mockCacheClearableProviderForModel) ClearCache() {
	m.cleared = true
}

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

	// /model new-model をシミュレート
	args := []string{"new-model"}
	result := handleModelCommand(agent, args)

	// コマンドが正常に処理されたことを確認
	assert.True(t, result)
	// モデルが切り替わったことを確認
	assert.Equal(t, "new-model", agent.CurrentModel)
	assert.Equal(t, "new-model", agent.session.Model)
	assert.Equal(t, "new-model", agent.Stats.Model)
	// ClearCacheが呼ばれたことを確認
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

func TestHandleProvidersCommand_UsesRuntimeOutput(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		ProviderName: "ollama",
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	result := handleProvidersCommand(agent)
	if !result {
		t.Fatal("handleProvidersCommand() = false, want true")
	}

	output := out.String()
	if !strings.Contains(output, "利用可能なプロバイダー") {
		t.Fatalf("expected runtime output to contain providers header, got %q", output)
	}
	if !strings.Contains(output, "/use <provider>") {
		t.Fatalf("expected runtime output to contain usage hint, got %q", output)
	}
	if !strings.Contains(output, "openai") {
		t.Fatalf("expected runtime output to contain provider list, got %q", output)
	}
}

func TestHandleProvidersCommand_MarksOnlyClaudeOwnerAsCurrent(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "claude",
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	if result := handleProvidersCommand(agent); !result {
		t.Fatal("handleProvidersCommand() = false, want true")
	}

	output := out.String()
	if strings.Count(output, "✓ ") != 1 {
		t.Fatalf("expected exactly one current marker, got output %q", output)
	}
	if !strings.Contains(output, "✓ claude") {
		t.Fatalf("expected claude to be marked current, got %q", output)
	}
	if strings.Contains(output, "✓ anthropic") {
		t.Fatalf("anthropic should not be marked current when claude owns the session, got %q", output)
	}
}

func TestHandleProvidersCommand_MarksOnlyAnthropicOwnerAsCurrent(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "anthropic",
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	if result := handleProvidersCommand(agent); !result {
		t.Fatal("handleProvidersCommand() = false, want true")
	}

	output := out.String()
	if strings.Count(output, "✓ ") != 1 {
		t.Fatalf("expected exactly one current marker, got output %q", output)
	}
	if !strings.Contains(output, "✓ anthropic") {
		t.Fatalf("expected anthropic to be marked current, got %q", output)
	}
	if strings.Contains(output, "✓ claude") {
		t.Fatalf("claude should not be marked current when anthropic owns the session, got %q", output)
	}
}

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
	if len(agent.History) != 0 {
		t.Fatalf("len(agent.History) = %d, want 0", len(agent.History))
	}
	if !strings.Contains(out.String(), "History cleared after provider switch") {
		t.Fatalf("expected output to contain history cleared notification, got %q", out.String())
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
