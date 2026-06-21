package agent

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// mockCacheClearableProvider はキャッシュクリアと Provider を実装したモック
type mockCacheClearableProvider struct {
	cleared          bool
	name             string
	responseID       string
	cachedResponseID bool
}

func (m *mockCacheClearableProvider) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock-cache"
}

func (m *mockCacheClearableProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	return "", nil
}

func (m *mockCacheClearableProvider) SupportsImages() bool {
	return false
}

func (m *mockCacheClearableProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return "", nil
}

func (m *mockCacheClearableProvider) IsFunctionCallingEnabled() bool {
	return false
}

func (m *mockCacheClearableProvider) ClearCache() {
	m.cleared = true
}

func (m *mockCacheClearableProvider) HasCachedResponseID() bool {
	return m.cachedResponseID
}

func (m *mockCacheClearableProvider) SetResponseID(id string) {
	m.responseID = id
	m.cachedResponseID = id != ""
}

func (m *mockCacheClearableProvider) GetResponseID() string {
	return m.responseID
}

func TestValidateProviderModelSelection_GeminiFunctionCalling(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	if err := validateProviderModelSelection(cfg, "gemini", "gemini", "gemini-3.5-flash", true); err != nil {
		t.Fatalf("supported Gemini model error = %v, want nil", err)
	}

	err := validateProviderModelSelection(cfg, "gemini", "gemini", "gemini-2.0-flash-lite", true)
	if err == nil || !strings.Contains(err.Error(), "gemini-3.1-flash-lite") {
		t.Fatalf("unsupported Gemini model error = %v, want replacement guidance", err)
	}
}

func TestAgent_SwitchProvider_ClearCache(t *testing.T) {
	// APIキーを一時的に設定（SwitchProviderのバリデーションを通過させるため）
	os.Setenv("OLLAMA_BASE_URL", "http://localhost:11434")
	defer os.Unsetenv("OLLAMA_BASE_URL")

	// テスト用にモックプロバイダーを登録
	api.RegisterProvider("ollama", func(apiKey string) (api.Provider, error) {
		return &mockCacheClearableProvider{}, nil
	})

	agent := &Agent{
		ProviderName: "mock",
		CurrentModel: "mock-model",
		Runtime:      NewAgentRuntimeWithConfig(newProjectMapDisabledConfig()),
		agentConversationState: agentConversationState{
			session: history.NewSession("mock-model"),
		},
	}

	mockProvider := &mockCacheClearableProvider{}
	agent.CurrentProvider = mockProvider

	// SwitchProviderを実行 (ollamaに切り替え)
	err := agent.SwitchProvider("ollama")

	assert.NoError(t, err)
	// ClearCacheが呼ばれたことを確認
	assert.True(t, mockProvider.cleared, "ClearCache should be called when switching provider")
	// プロバイダーが切り替わったことを確認
	assert.Equal(t, "ollama", agent.ProviderName)
	assert.NotEqual(t, mockProvider, agent.CurrentProvider, "CurrentProvider should be replaced")
	assert.Equal(t, agent.CurrentModel, agent.session.Model)
}

func TestAgent_SwitchProvider_KeepsContextClearsRemoteContinuationAndNotifies(t *testing.T) {
	// APIキー存在チェックのため、OLLAMA_BASE_URL を用意
	t.Setenv("OLLAMA_BASE_URL", "http://localhost:11434")

	// テスト用にモックプロバイダーを登録
	api.RegisterProvider("ollama", func(apiKey string) (api.Provider, error) {
		return &mockCacheClearableProvider{}, nil
	})

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(newProjectMapDisabledConfig())
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, &out)

	agent := &Agent{
		ProviderName:    "mock",
		CurrentModel:    "mock-model",
		CurrentProvider: &mockCacheClearableProvider{responseID: "resp_old", cachedResponseID: true},
		Runtime:         runtime,
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
		agentConversationState: agentConversationState{
			session: history.NewSession("mock-model"),
		},
	}
	agent.session.AddMessage("user", "old task", agent.CurrentModel)
	agent.session.CompactedItems = []history.CompactedItem{{Type: "compacted", Data: "compressed"}}
	agent.session.IsCompactedMode = true
	agent.session.ApplyResponseContext("resp_old", "mock-model", "mock", "mock")
	agent.persistSession()

	err := agent.SwitchProvider("ollama")
	assert.NoError(t, err)

	assert.Equal(t, 2, len(agent.History))
	assert.Len(t, agent.session.Messages, 1)
	assert.True(t, agent.session.IsCompactedMode)
	assert.Equal(t, []history.CompactedItem{{Type: "compacted", Data: "compressed"}}, agent.session.CompactedItems)
	assert.Empty(t, agent.session.ResponseID)
	assert.Empty(t, agent.session.ResponseModel)
	assert.Empty(t, agent.session.ResponseProviderName)
	assert.Empty(t, agent.session.ResponseProviderConfigKey)
	assert.Contains(t, out.String(), "Context kept locally; provider remote continuation reset")
	assert.NotContains(t, out.String(), "History cleared after provider switch")
}

func TestAgent_SwitchProvider_RebuildsSystemPrompt(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "test-key")

	cfg := newProjectMapDisabledConfig()
	runtime := NewAgentRuntimeWithConfig(cfg)

	agent := &Agent{
		ProviderName:    "gemini",
		CurrentModel:    "gemini-3.1-pro-preview-customtools",
		CurrentProvider: &mockCacheClearableProvider{name: "gemini"},
		SystemPrompt:    prompt.BuildProviderSystemPromptWithConfig(prompt.SystemPrompt, "gemini", "gemini-3.1-pro-preview-customtools", cfg),
		Stats:           NewSessionStats("gemini"),
		Runtime:         runtime,
	}

	err := agent.SwitchProvider("deepseek")
	assert.NoError(t, err)
	assert.NotContains(t, agent.SystemPrompt, "## Provider Notes")
	assert.Contains(t, agent.SystemPrompt, "### Legacy edit tools")
	assert.NotContains(t, agent.SystemPrompt, "### apply_patch (edit tool)")
}

func TestAgent_SwitchProvider_ResetsCostUnknownState(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	api.RegisterProvider("openai", func(apiKey string) (api.Provider, error) {
		return &mockCacheClearableProvider{name: "openai"}, nil
	})

	cfg := newProjectMapDisabledConfig()
	runtime := NewAgentRuntimeWithConfig(cfg)
	stats := NewSessionStats("bedrock", "amazon.nova-pro-v1:0")
	stats.InputTokens = 1000
	stats.OutputTokens = 200
	stats.AccumulatedCost = 0.123
	stats.CostUnknown = true
	stats.CostUnknownEvents = 1
	stats.LastTurnCostUnknown = true
	stats.LastUsage = &api.Usage{InputTokens: 1000, OutputTokens: 200}
	stats.LastTurnUsage = &api.Usage{InputTokens: 1000, OutputTokens: 200}
	stats.ToolExecutions["read_file"] = 2

	agent := &Agent{
		ProviderName:    "bedrock",
		CurrentModel:    "amazon.nova-pro-v1:0",
		CurrentProvider: &mockCacheClearableProvider{name: "bedrock"},
		Stats:           stats,
		Runtime:         runtime,
		agentConversationState: agentConversationState{
			session: history.NewSession("amazon.nova-pro-v1:0"),
		},
	}

	err := agent.SwitchProvider("openai")
	assert.NoError(t, err)
	assert.Equal(t, "openai", agent.ProviderName)
	assert.False(t, agent.Stats.CostUnknown)
	assert.Equal(t, 0, agent.Stats.CostUnknownEvents)
	assert.Equal(t, 0.0, agent.Stats.AccumulatedCost)
	assert.False(t, agent.Stats.LastTurnCostUnknown)
	assert.Nil(t, agent.Stats.LastUsage)
	assert.Nil(t, agent.Stats.LastTurnUsage)
	assert.Equal(t, 0, agent.Stats.InputTokens)
	assert.Equal(t, 0, len(agent.Stats.ToolExecutions))

	estimate := agent.Stats.EstimatedCostEstimateForConfig(agent.cfg())
	assert.False(t, estimate.PricingUnavailable)
}

func TestAgent_SwitchProvider_UsesProviderDefaultWhenOtherProviderHasOverride(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("OPENAI_API_KEY", "test-key")

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
	cfg.ProjectMap.Enabled = false

	agent := &Agent{
		ProviderName:    "deepseek",
		CurrentModel:    "deepseek-custom",
		CurrentProvider: &mockCacheClearableProvider{name: "deepseek"},
		Runtime:         NewAgentRuntimeWithConfig(cfg),
		agentConversationState: agentConversationState{
			session: history.NewSession("deepseek-custom"),
		},
	}

	err = agent.SwitchProvider("openai")
	assert.NoError(t, err)
	assert.Equal(t, "openai", agent.ProviderName)
	want := config.DefaultConfig().ProviderModels["openai"].DefaultModel
	assert.Equal(t, want, agent.CurrentModel)
	assert.Equal(t, want, agent.session.Model)
}

func TestAgent_SwitchProvider_DefaultModelWinsWhenExplicitEntryHasOnlyNonModelSettings(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "gpt-custom"
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		MaxOutputTokens: 99999,
	})

	agent := &Agent{
		ProviderName:    "deepseek",
		CurrentModel:    "deepseek-chat",
		CurrentProvider: &mockCacheClearableProvider{name: "deepseek"},
		Runtime:         NewAgentRuntimeWithConfig(cfg),
		agentConversationState: agentConversationState{
			session: history.NewSession("deepseek-chat"),
		},
	}

	err := agent.SwitchProvider("openai")
	assert.NoError(t, err)
	assert.Equal(t, "openai", agent.ProviderName)
	assert.Equal(t, "gpt-custom", agent.CurrentModel)
	assert.Equal(t, "gpt-custom", agent.session.Model)
}

func TestAgent_SwitchProvider_KeepsConfiguredOllamaModelThatLooksLikeAnotherProvider(t *testing.T) {
	t.Setenv("OLLAMA_BASE_URL", "http://localhost:11434")

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "ollama"
	cfg.DefaultModel = "deepseek-r1:8b"

	agent := &Agent{
		ProviderName:    "deepseek",
		CurrentModel:    "deepseek-chat",
		CurrentProvider: &mockCacheClearableProvider{name: "deepseek"},
		Runtime:         NewAgentRuntimeWithConfig(cfg),
		agentConversationState: agentConversationState{
			session: history.NewSession("deepseek-chat"),
		},
	}

	err := agent.SwitchProvider("ollama")
	assert.NoError(t, err)
	assert.Equal(t, "ollama", agent.ProviderName)
	assert.Equal(t, "deepseek-r1:8b", agent.CurrentModel)
	assert.Equal(t, "deepseek-r1:8b", agent.session.Model)
}

func TestAgent_SwitchProvider_CanonicalizesAnthropicAlias(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.DefaultModel = "claude-custom"

	agent := &Agent{
		ProviderName:    "deepseek",
		CurrentModel:    "deepseek-chat",
		CurrentProvider: &mockCacheClearableProvider{name: "deepseek"},
		Runtime:         NewAgentRuntimeWithConfig(cfg),
		agentConversationState: agentConversationState{
			session: history.NewSession("deepseek-chat"),
		},
	}

	err := agent.SwitchProvider("anthropic")
	assert.NoError(t, err)
	assert.Equal(t, "claude", agent.ProviderName)
	assert.Equal(t, "claude-custom", agent.CurrentModel)
	assert.Equal(t, "claude-custom", agent.session.Model)
	if agent.CurrentProvider == nil {
		t.Fatal("CurrentProvider should not be nil")
	}
	assert.Equal(t, "claude", config.CanonicalProviderName(agent.CurrentProvider.Name()))
}

func TestAgent_SwitchProvider_AnthropicAliasUsesAliasSpecificModelWhenBothKeysExist(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: deepseek
provider_models:
  anthropic:
    default_model: anthropic-custom
  claude:
    default_model: claude-custom
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ProjectMap.Enabled = false

	agent := &Agent{
		ProviderName:    "deepseek",
		CurrentModel:    "deepseek-chat",
		CurrentProvider: &mockCacheClearableProvider{name: "deepseek"},
		Runtime:         NewAgentRuntimeWithConfig(cfg),
		agentConversationState: agentConversationState{
			session: history.NewSession("deepseek-chat"),
		},
	}

	err = agent.SwitchProvider("anthropic")
	assert.NoError(t, err)
	assert.Equal(t, "claude", agent.ProviderName)
	assert.Equal(t, "anthropic-custom", agent.CurrentModel)
	assert.Equal(t, "anthropic-custom", agent.session.Model)
}
