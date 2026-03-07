package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
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
	agent := &Agent{
		ProviderName: "mock",
		CurrentModel: "old-model",
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
	// ClearCacheが呼ばれたことを確認
	assert.True(t, mockProvider.cleared, "ClearCache should be called when switching model")
}

func TestHandleModelCommand_RebuildsClaudePromptForOpus(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(nil)
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	agent := &Agent{
		ProviderName:    "claude",
		CurrentModel:    "claude-sonnet-4-6",
		CurrentProvider: &mockCacheClearableProviderForModel{name: "claude"},
		SystemPrompt:    prompt.BuildProviderSystemPrompt(prompt.SystemPrompt, "claude", "claude-sonnet-4-6"),
	}

	result := handleModelCommand(agent, []string{"claude-opus-4-6"})

	assert.True(t, result)
	assert.Contains(t, agent.SystemPrompt, "### Stable Working Reference")
	assert.Contains(t, agent.SystemPrompt, "### Claude-specific")
}
