package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

func TestNewDeepSeekProvider(t *testing.T) {
	apiKey := "test-api-key"
	provider := NewDeepSeekProvider(apiKey)

	if provider == nil {
		t.Fatal("NewDeepSeekProvider() returned nil")
	}
}

func TestNewDeepSeekProvider_URLOverride(t *testing.T) {
	// URL環境変数を保存して後でリストア
	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)

	t.Run("DefaultURL", func(t *testing.T) {
		os.Unsetenv("DEEPSEEK_API_URL")
		p := NewDeepSeekProvider("test-key")
		if p.apiURL != defaultDeepSeekURL {
			t.Errorf("apiURL = %q, want %q", p.apiURL, defaultDeepSeekURL)
		}
	})

	t.Run("CustomURL", func(t *testing.T) {
		customURL := "https://custom.deepseek.api.com/v1"
		os.Setenv("DEEPSEEK_API_URL", customURL)
		p := NewDeepSeekProvider("test-key")
		if p.apiURL != customURL {
			t.Errorf("apiURL = %q, want %q", p.apiURL, customURL)
		}
	})
}

func TestDeepSeekProvider_Name(t *testing.T) {
	provider := NewDeepSeekProvider("test-key")

	name := provider.Name()
	if name != "DeepSeek" {
		t.Errorf("Name() = %v, want 'DeepSeek'", name)
	}
}

func TestDeepSeekProvider_SupportsImages(t *testing.T) {
	provider := NewDeepSeekProvider("test-key")

	supports := provider.SupportsImages()
	if supports {
		t.Error("SupportsImages() = true, want false (DeepSeek does not support images)")
	}
}

func TestGetActualModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  string
	}{
		{
			name:  "deepseek-chat (default)",
			model: "deepseek-chat",
			want:  "deepseek-chat",
		},
		{
			name:  "deepseek-coder",
			model: "deepseek-coder",
			want:  "deepseek-coder",
		},
		{
			name:  "deepseek-reasoner",
			model: "deepseek-reasoner",
			want:  "deepseek-reasoner",
		},
		{
			name:  "empty string",
			model: "",
			want:  "deepseek-chat", // デフォルト
		},
		{
			name:  "unknown model",
			model: "unknown-model",
			want:  "deepseek-chat", // デフォルト
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getActualModel(tt.model)
			if got != tt.want {
				t.Errorf("getActualModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestDeepSeekProvider_ChatWithTools_RequestValidation(t *testing.T) {
	// モックサーバーを作成（リクエスト検証用）
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequestMethod(t, r, "POST")
		assertJSONContentType(t, r)
		assertRequestHeader(t, r, "Authorization", "Bearer test-key")

		// リクエストボディを検証
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if req.Model != "deepseek-chat" {
			t.Errorf("Model = %q, want 'deepseek-chat'", req.Model)
		}
		if len(req.Messages) != 2 { // system + user
			t.Errorf("Messages count = %d, want 2", len(req.Messages))
		}
		if !req.Stream {
			t.Error("Stream should be true")
		}

		// ストリーミングレスポンス
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		chunks := []string{
			`{"choices":[{"delta":{"content":"Test"}}]}`,
			`{"choices":[{"delta":{"content":" response"}}]}`,
		}
		for _, chunk := range chunks {
			_, _ = w.Write([]byte("data: " + chunk + "\n\n"))
			flusher.Flush()
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	})

	// 環境変数でモックサーバーURLを設定
	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := NewDeepSeekProvider("test-key")
	history := []Message{{Role: "user", Content: "Hello"}}

	result, err := p.ChatWithTools(context.Background(), "System prompt", history, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Test response" {
		t.Errorf("ChatWithTools() = %q, want 'Test response'", result)
	}
}

func TestDeepSeekProvider_ChatWithTools_Streaming(t *testing.T) {
	chunks := []string{
		`{"choices":[{"delta":{"content":"Hello"}}]}`,
		`{"choices":[{"delta":{"content":" World"}}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := NewDeepSeekProvider("test-key")
	history := []Message{{Role: "user", Content: "Hi"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "deepseek-chat")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Hello World" {
		t.Errorf("ChatWithTools() = %q, want 'Hello World'", result)
	}
}

func TestDeepSeekProvider_ChatWithTools_ServerError(t *testing.T) {
	server := mockAPIServer(t, errorHandler(500, "Internal Server Error"))

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := NewDeepSeekProvider("test-key")
	history := []Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for 500 status")
	}
}

func TestDeepSeekProvider_ChatWithTools_RateLimitResponse(t *testing.T) {
	server := mockAPIServer(t, rateLimitHandler("60"))

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := NewDeepSeekProvider("test-key")
	history := []Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for rate limit")
	}
}

func TestDeepSeekProvider_ChatWithTools_ContextCancel(t *testing.T) {
	// 長時間待機するハンドラー
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // コンテキストがキャンセルされるまで待機
	})

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := NewDeepSeekProvider("test-key")
	history := []Message{{Role: "user", Content: "Hi"}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 即座にキャンセル

	_, err := p.ChatWithTools(ctx, "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error when context is cancelled")
	}
}

func TestDeepSeekProvider_ChatWithImage_NoImage(t *testing.T) {
	chunks := []string{
		`{"choices":[{"delta":{"content":"No image"}}]}`,
		`{"choices":[{"delta":{"content":" response"}}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := NewDeepSeekProvider("test-key")

	// imageがnilの場合
	result, err := p.ChatWithImage(context.Background(), "System", nil, "Hello", nil, "")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "No image response" {
		t.Errorf("ChatWithImage() = %q, want 'No image response'", result)
	}
}

func TestDeepSeekProvider_ChatWithImage_WithImage(t *testing.T) {
	chunks := []string{
		`{"choices":[{"delta":{"content":"Image ignored"}}]}`,
		`{"choices":[{"delta":{"content":" response"}}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := NewDeepSeekProvider("test-key")

	// imageがある場合（DeepSeekは画像非対応なので警告を出してテキストのみ送信）
	image := &ImageData{Base64: "dGVzdA==", MediaType: "image/png"}
	result, err := p.ChatWithImage(context.Background(), "System", nil, "Describe this", image, "")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "Image ignored response" {
		t.Errorf("ChatWithImage() = %q, want 'Image ignored response'", result)
	}
}
