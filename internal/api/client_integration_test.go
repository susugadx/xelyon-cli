package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/deepseek"
)

func TestDeepSeekProvider_ChatWithTools_Integration(t *testing.T) {
	t.Skip("Streaming response mock needs proper implementation")
	// DeepSeek APIをモック
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// リクエスト検証
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if !strings.Contains(r.Header.Get("Authorization"), "Bearer") {
			t.Error("Authorization header missing or invalid")
		}

		// モックレスポンス
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := `{
			"id": "test-id",
			"choices": [{
				"message": {
					"role": "assistant",
					"content": "Test response"
				},
				"finish_reason": "stop"
			}],
			"usage": {
				"prompt_tokens": 10,
				"completion_tokens": 5,
				"total_tokens": 15
			}
		}`
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	// DeepSeekProviderを作成してapiURLを上書き
	os.Setenv("DEEPSEEK_API_URL", server.URL)
	defer os.Unsetenv("DEEPSEEK_API_URL")

	provider := api.NewDeepSeekProvider("test-api-key")

	history := []api.Message{
		{Role: "user", Content: "Hello"},
	}

	ctx := context.Background()
	response, err := provider.ChatWithTools(ctx, "system prompt", history, "deepseek-chat")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	if !strings.Contains(response, "Test response") {
		t.Errorf("ChatWithTools() response = %q, should contain 'Test response'", response)
	}
}

func TestDeepSeekProvider_ChatWithTools_NetworkError(t *testing.T) {
	// 無効なURLでプロバイダー作成
	os.Setenv("DEEPSEEK_API_URL", "http://invalid-host-that-does-not-exist:99999")
	defer os.Unsetenv("DEEPSEEK_API_URL")

	provider := api.NewDeepSeekProvider("test-api-key")

	history := []api.Message{
		{Role: "user", Content: "Hello"},
	}

	ctx := context.Background()
	_, err := provider.ChatWithTools(ctx, "system prompt", history, "deepseek-chat")
	if err == nil {
		t.Error("ChatWithTools() should return error for network failure")
	}
}

func TestDeepSeekProvider_ChatWithTools_APIError(t *testing.T) {
	// エラーレスポンスを返すサーバー
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"message": "Invalid request"}}`))
	}))
	defer server.Close()

	os.Setenv("DEEPSEEK_API_URL", server.URL)
	defer os.Unsetenv("DEEPSEEK_API_URL")

	provider := api.NewDeepSeekProvider("test-api-key")

	history := []api.Message{
		{Role: "user", Content: "Hello"},
	}

	ctx := context.Background()
	_, err := provider.ChatWithTools(ctx, "system prompt", history, "deepseek-chat")
	if err == nil {
		t.Error("ChatWithTools() should return error for API error response")
	}
}

func TestDeepSeekProvider_ChatWithTools_RateLimit(t *testing.T) {
	// 429レスポンスを返すサーバー
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": {"message": "Rate limit exceeded"}}`))
	}))
	defer server.Close()

	os.Setenv("DEEPSEEK_API_URL", server.URL)
	defer os.Unsetenv("DEEPSEEK_API_URL")

	provider := api.NewDeepSeekProvider("test-api-key")

	history := []api.Message{
		{Role: "user", Content: "Hello"},
	}

	ctx := context.Background()
	_, err := provider.ChatWithTools(ctx, "system prompt", history, "deepseek-chat")
	if err == nil {
		t.Error("ChatWithTools() should return error for rate limit")
	}
	if err != nil && !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("ChatWithTools() error should mention rate limit, got: %v", err)
	}
}

func TestDeepSeekProvider_ChatWithImage_NotSupported(t *testing.T) {
	t.Skip("ChatWithImage calls real API, needs mock server")
	// 画像非対応プロバイダー（DeepSeek）
	provider := api.NewDeepSeekProvider("test-api-key")

	history := []api.Message{
		{Role: "user", Content: "Hello"},
	}

	imageData := &api.ImageData{
		Base64:    "fake-image-data",
		MediaType: "image/png",
	}

	ctx := context.Background()
	_, err := provider.ChatWithImage(ctx, "system prompt", history, "What's in this image?", imageData, "deepseek-chat")
	if err == nil {
		t.Error("ChatWithImage() should return error for unsupported provider")
	}
	if err != nil && !strings.Contains(err.Error(), "does not support") {
		t.Errorf("ChatWithImage() error should mention unsupported, got: %v", err)
	}
}
