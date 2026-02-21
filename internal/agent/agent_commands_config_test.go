package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/susugadx/xelyon-cli/internal/api"
)

// mockCacheClearableProviderForModel はキャッシュクリアと Provider を実装したモック（モデルコマンド用）
type mockCacheClearableProviderForModel struct {
	cleared bool
}

func (m *mockCacheClearableProviderForModel) Name() string {
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
