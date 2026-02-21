package agent

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/susugadx/xelyon-cli/internal/api"
)

// mockCacheClearableProvider はキャッシュクリアと Provider を実装したモック
type mockCacheClearableProvider struct {
	cleared bool
}

func (m *mockCacheClearableProvider) Name() string {
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
}
