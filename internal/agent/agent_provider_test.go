package agent

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// mockCacheClearableProvider はキャッシュクリアと Provider を実装したモック
type mockCacheClearableProvider struct {
	cleared bool
	name    string
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

func TestAgent_SwitchProvider_ClearHistoryAndNotify(t *testing.T) {
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
		CurrentProvider: &mockCacheClearableProvider{},
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

	err := agent.SwitchProvider("ollama")
	assert.NoError(t, err)

	assert.Equal(t, 0, len(agent.History))
	assert.Contains(t, out.String(), "History cleared after provider switch")
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
	assert.Contains(t, agent.SystemPrompt, "### DeepSeek-specific")
	assert.NotContains(t, agent.SystemPrompt, "### Gemini-specific")
}
