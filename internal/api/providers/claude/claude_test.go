package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestNewClaudeProvider(t *testing.T) {
	apiKey := "test-api-key"
	provider := New(apiKey)

	if provider == nil {
		t.Fatal("New() returned nil")
	}
}

func TestClaudeProvider_Name(t *testing.T) {
	provider := New("test-key")

	name := provider.Name()
	if name != "Claude" {
		t.Errorf("Name() = %v, want 'Claude'", name)
	}
}

func TestClaudeProvider_SupportsImages(t *testing.T) {
	provider := New("test-key")

	supports := provider.SupportsImages()
	if !supports {
		t.Error("SupportsImages() = false, want true (Claude supports images)")
	}
}

func TestNewClaudeProvider_URLOverride(t *testing.T) {
	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)

	t.Run("DefaultURL", func(t *testing.T) {
		os.Unsetenv("ANTHROPIC_API_URL")
		p := New("test-key")
		if p.apiURL != defaultClaudeURL {
			t.Errorf("apiURL = %q, want %q", p.apiURL, defaultClaudeURL)
		}
	})

	t.Run("CustomURL", func(t *testing.T) {
		customURL := "https://custom.anthropic.api.com/v1"
		os.Setenv("ANTHROPIC_API_URL", customURL)
		p := New("test-key")
		if p.apiURL != customURL {
			t.Errorf("apiURL = %q, want %q", p.apiURL, customURL)
		}
	})
}

// mockAPIServer creates a test HTTP server
func mockAPIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// claudeStreamingHandler はClaude形式のストリーミングハンドラー
func claudeStreamingHandler(texts []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		for _, text := range texts {
			event := StreamEvent{
				Type:  "content_block_delta",
				Delta: Delta{Type: "text_delta", Text: text},
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}

		// 終了イベント
		stopEvent := StreamEvent{Type: "message_stop"}
		data, _ := json.Marshal(stopEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

// errorHandler returns a handler that responds with error
func errorHandler(statusCode int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		_, _ = fmt.Fprintf(w, `{"error":{"message":"%s"}}`, message)
	}
}

// rateLimitHandler returns a handler that responds with rate limit error
func rateLimitHandler(retryAfter string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", retryAfter)
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limit exceeded"}}`))
	}
}

func TestClaudeProvider_ChatWithTools_NonStreaming(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("x-api-key = %q, want 'test-key'", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("anthropic-version = %q, want '2023-06-01'", r.Header.Get("anthropic-version"))
		}

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if req.Model != "claude-sonnet-4-20250514" {
			t.Errorf("Model = %q, want 'claude-sonnet-4-20250514'", req.Model)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := Response{
			Content: []Content{{Type: "text", Text: "Test response from Claude"}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	result, err := p.ChatWithTools(context.Background(), "System prompt", history, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Test response from Claude" {
		t.Errorf("ChatWithTools() = %q, want 'Test response from Claude'", result)
	}
}

func TestClaudeProvider_ChatWithTools_Streaming(t *testing.T) {
	server := mockAPIServer(t, claudeStreamingHandler([]string{"Hello", " from", " Claude"}))

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Hello from Claude" {
		t.Errorf("ChatWithTools() = %q, want 'Hello from Claude'", result)
	}
}

func TestClaudeProvider_ChatWithTools_APIError(t *testing.T) {
	server := mockAPIServer(t, errorHandler(500, "Internal Server Error"))

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for 500 status")
	}
}

func TestClaudeProvider_ChatWithTools_RateLimit(t *testing.T) {
	server := mockAPIServer(t, rateLimitHandler("60"))

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for rate limit")
	}
}

func TestClaudeProvider_ChatWithImage_NoImage(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := Response{
			Content: []Content{{Type: "text", Text: "No image response"}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")

	result, err := p.ChatWithImage(context.Background(), "System", nil, "Hello", nil, "")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "No image response" {
		t.Errorf("ChatWithImage() = %q, want 'No image response'", result)
	}
}

func TestClaudeProvider_ChatWithImage_WithImage(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		// リクエストボディにimageソースがあることを確認
		var req MultimodalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := Response{
			Content: []Content{{Type: "text", Text: "Image analysis complete"}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png"}

	result, err := p.ChatWithImage(context.Background(), "System", nil, "Describe this", image, "")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "Image analysis complete" {
		t.Errorf("ChatWithImage() = %q, want 'Image analysis complete'", result)
	}
}

func TestLevelToBudgetTokens(t *testing.T) {
	tests := []struct {
		level string
		want  int
	}{
		{"low", 5000},
		{"medium", 10000},
		{"high", 20000},
		{"xhigh", 40000},
		{"unknown", 10000}, // default
		{"", 10000},        // default
		{"invalid", 10000}, // default
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			got := LevelToBudgetTokens(tt.level)
			if got != tt.want {
				t.Errorf("LevelToBudgetTokens(%q) = %d, want %d", tt.level, got, tt.want)
			}
		})
	}
}
