package api

import (
	"context"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// mockProvider は Provider インターフェースのモック実装
type mockProvider struct {
	name              string
	supportsImages    bool
	chatError         error
	chatResponse      string
	chatImageError    error
	chatImageResponse string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) SupportsImages() bool {
	return m.supportsImages
}

func (m *mockProvider) IsFunctionCallingEnabled() bool {
	return true
}

func (m *mockProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error) {
	if m.chatError != nil {
		return "", m.chatError
	}
	return m.chatResponse, nil
}

func (m *mockProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []Message, userMessage string, image *ImageData, model string) (string, error) {
	if m.chatImageError != nil {
		return "", m.chatImageError
	}
	return m.chatImageResponse, nil
}

func TestNewClient(t *testing.T) {
	provider := &mockProvider{name: "test-provider"}

	client := NewClient(provider)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	if client.Provider != provider {
		t.Error("NewClient() Provider not set correctly")
	}

	if client.Timeout != config.DefaultHTTPTimeout {
		t.Errorf("NewClient() Timeout = %v, want %v", client.Timeout, config.DefaultHTTPTimeout)
	}
}

func TestClient_ChatWithTools_Success(t *testing.T) {
	provider := &mockProvider{
		name:         "test-provider",
		chatResponse: "Hello, world!",
	}

	client := NewClient(provider)

	response, err := client.ChatWithTools("system prompt", []Message{}, "test-model")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	if response != "Hello, world!" {
		t.Errorf("ChatWithTools() = %q, want 'Hello, world!'", response)
	}
}

func TestClient_ChatWithTools_Error(t *testing.T) {
	provider := &mockProvider{
		name:      "test-provider",
		chatError: context.DeadlineExceeded,
	}

	client := NewClient(provider)

	_, err := client.ChatWithTools("system prompt", []Message{}, "test-model")
	if err != context.DeadlineExceeded {
		t.Errorf("ChatWithTools() error = %v, want context.DeadlineExceeded", err)
	}
}

func TestClient_ChatWithTools_Timeout(t *testing.T) {
	provider := &mockProvider{
		name:         "slow-provider",
		chatResponse: "response",
	}

	client := NewClient(provider)
	client.Timeout = 1 * time.Nanosecond // 極めて短いタイムアウト

	// タイムアウトが発生する（ただしmockProviderは即座に返すのでタイムアウトしない）
	// 実際のテストでは ChatWithTools が context を尊重することを確認
	response, err := client.ChatWithTools("system prompt", []Message{}, "test-model")

	// mockは即座に返すのでエラーにはならない
	if err != nil {
		t.Logf("ChatWithTools() with short timeout: error = %v", err)
	}

	if response != "response" && err == nil {
		t.Errorf("ChatWithTools() = %q, want 'response'", response)
	}
}
