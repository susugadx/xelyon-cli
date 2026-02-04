package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"

	// ツール登録のための blank import
	_ "github.com/susugadx/xelyon-cli/internal/tools/dev"
	_ "github.com/susugadx/xelyon-cli/internal/tools/file"
	_ "github.com/susugadx/xelyon-cli/internal/tools/git"
	_ "github.com/susugadx/xelyon-cli/internal/tools/lsp"
	_ "github.com/susugadx/xelyon-cli/internal/tools/search"
)

// mockAPIServer creates a test HTTP server
func mockAPIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// streamingHandler returns a handler that sends SSE events
func streamingHandler(chunks []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}

		fmt.Fprintf(w, "data: [DONE]\n\n")
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

func TestNewOpenAIProvider(t *testing.T) {
	apiKey := "test-api-key"
	provider := New(apiKey)

	if provider == nil {
		t.Fatal("New() returned nil")
	}
}

func TestOpenAIProvider_Name(t *testing.T) {
	provider := New("test-key")

	name := provider.Name()
	if name != "OpenAI" {
		t.Errorf("Name() = %v, want 'OpenAI'", name)
	}
}

func TestOpenAIProvider_SupportsImages(t *testing.T) {
	provider := New("test-key")

	supports := provider.SupportsImages()
	if !supports {
		t.Error("SupportsImages() = false, want true (OpenAI supports images)")
	}
}

func TestNewOpenAIProvider_URLOverride(t *testing.T) {
	originalURL := os.Getenv("OPENAI_API_URL")
	defer os.Setenv("OPENAI_API_URL", originalURL)

	t.Run("DefaultURL", func(t *testing.T) {
		os.Unsetenv("OPENAI_API_URL")
		p := New("test-key")
		if p.apiURL != defaultOpenAIURL {
			t.Errorf("apiURL = %q, want %q", p.apiURL, defaultOpenAIURL)
		}
	})

	t.Run("CustomURL", func(t *testing.T) {
		customURL := "https://custom.openai.api.com/v1"
		os.Setenv("OPENAI_API_URL", customURL)
		p := New("test-key")
		if p.apiURL != customURL {
			t.Errorf("apiURL = %q, want %q", p.apiURL, customURL)
		}
	})
}

func TestOpenAIProvider_ChatWithTools_NonStreaming(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization = %q, want 'Bearer test-key'", r.Header.Get("Authorization"))
		}

		var req api.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if req.Model != "gpt-4o" {
			t.Errorf("Model = %q, want 'gpt-4o'", req.Model)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := api.ChatResponse{
			Choices: []api.Choice{
				{Message: api.Message{Content: "Test response from OpenAI"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("OPENAI_API_URL")
	defer os.Setenv("OPENAI_API_URL", originalURL)
	os.Setenv("OPENAI_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	result, err := p.ChatWithTools(context.Background(), "System prompt", history, "gpt-4o")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Test response from OpenAI" {
		t.Errorf("ChatWithTools() = %q, want 'Test response from OpenAI'", result)
	}
}

func TestOpenAIProvider_ChatWithTools_Streaming(t *testing.T) {
	chunks := []string{
		`{"choices":[{"delta":{"content":"Hello"}}]}`,
		`{"choices":[{"delta":{"content":" from"}}]}`,
		`{"choices":[{"delta":{"content":" OpenAI"}}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("OPENAI_API_URL")
	defer os.Setenv("OPENAI_API_URL", originalURL)
	os.Setenv("OPENAI_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "gpt-4o")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Hello from OpenAI" {
		t.Errorf("ChatWithTools() = %q, want 'Hello from OpenAI'", result)
	}
}

func TestOpenAIProvider_ChatWithTools_APIError(t *testing.T) {
	server := mockAPIServer(t, errorHandler(500, "Internal Server Error"))

	originalURL := os.Getenv("OPENAI_API_URL")
	defer os.Setenv("OPENAI_API_URL", originalURL)
	os.Setenv("OPENAI_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for 500 status")
	}
}

func TestOpenAIProvider_ChatWithTools_RateLimit(t *testing.T) {
	server := mockAPIServer(t, rateLimitHandler("60"))

	originalURL := os.Getenv("OPENAI_API_URL")
	defer os.Setenv("OPENAI_API_URL", originalURL)
	os.Setenv("OPENAI_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for rate limit")
	}
}

func TestOpenAIProvider_ChatWithImage_NoImage(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := api.ChatResponse{
			Choices: []api.Choice{
				{Message: api.Message{Content: "No image response"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("OPENAI_API_URL")
	defer os.Setenv("OPENAI_API_URL", originalURL)
	os.Setenv("OPENAI_API_URL", server.URL)

	// ChatWithImage は image=nil の場合 ChatWithTools 経由になるが、
	// モデル判定で Responses API 側に流れてもモックを使えるように URL も上書きしておく
	originalResponsesURL := os.Getenv("OPENAI_RESPONSES_URL")
	defer os.Setenv("OPENAI_RESPONSES_URL", originalResponsesURL)
	os.Setenv("OPENAI_RESPONSES_URL", server.URL)

	p := New("test-key")

	result, err := p.ChatWithImage(context.Background(), "System", nil, "Hello", nil, "gpt-4o")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "No image response" {
		t.Errorf("ChatWithImage() = %q, want 'No image response'", result)
	}
}

func TestOpenAIProvider_ChatWithImage_WithImage(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		// リクエストボディにimage_urlがあることを確認
		var req MultimodalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if len(req.Messages) < 2 { // system + multimodal message
			t.Errorf("Messages count = %d, want at least 2", len(req.Messages))
		}

		w.Header().Set("Content-Type", "application/json")
		resp := api.ChatResponse{
			Choices: []api.Choice{
				{Message: api.Message{Content: "Image analysis complete"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("OPENAI_API_URL")
	defer os.Setenv("OPENAI_API_URL", originalURL)
	os.Setenv("OPENAI_API_URL", server.URL)

	// Responses API 側に流れてもモックを使えるように URL も上書きしておく
	originalResponsesURL := os.Getenv("OPENAI_RESPONSES_URL")
	defer os.Setenv("OPENAI_RESPONSES_URL", originalResponsesURL)
	os.Setenv("OPENAI_RESPONSES_URL", server.URL)

	p := New("test-key")
	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png"}

	result, err := p.ChatWithImage(context.Background(), "System", nil, "Describe this", image, "gpt-4o")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "Image analysis complete" {
		t.Errorf("ChatWithImage() = %q, want 'Image analysis complete'", result)
	}
}

// --- Responses API テスト ---

func TestOpenAIProvider_ResponsesAPI_Streaming(t *testing.T) {
	// Responses API ストリーミング形式のチャンク
	chunks := []string{
		`{"type":"response.output_text.delta","delta":"Hello"}`,
		`{"type":"response.output_text.delta","delta":" from"}`,
		`{"type":"response.output_text.delta","delta":" Codex"}`,
		`{"type":"response.completed"}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("OPENAI_RESPONSES_URL")
	defer os.Setenv("OPENAI_RESPONSES_URL", originalURL)
	os.Setenv("OPENAI_RESPONSES_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	result, err := p.chatWithResponses(context.Background(), "System", history, "gpt-5.2-codex")
	if err != nil {
		t.Fatalf("chatWithResponses() error = %v", err)
	}
	if result != "Hello from Codex" {
		t.Errorf("chatWithResponses() = %q, want 'Hello from Codex'", result)
	}
}

func TestOpenAIProvider_ResponsesAPI_RequestFormat(t *testing.T) {
	var receivedReq ResponsesRequest

	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Method = %q, want POST", r.Method)
		}

		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// 非ストリーミングレスポンス
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"output": []map[string]interface{}{
				{
					"type": "message",
					"content": []map[string]interface{}{
						{"type": "output_text", "text": "Response"},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("OPENAI_RESPONSES_URL")
	defer os.Setenv("OPENAI_RESPONSES_URL", originalURL)
	os.Setenv("OPENAI_RESPONSES_URL", server.URL)

	p := New("test-key")
	history := []api.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
	}

	_, _ = p.chatWithResponses(context.Background(), "You are helpful", history, "gpt-5.2-codex")

	// リクエスト形式を確認
	if receivedReq.Model != "gpt-5.2-codex" {
		t.Errorf("Model = %q, want 'gpt-5.2-codex'", receivedReq.Model)
	}
	if receivedReq.Instructions != "You are helpful" {
		t.Errorf("Instructions = %q, want 'You are helpful'", receivedReq.Instructions)
	}
	if !receivedReq.Stream {
		t.Error("Stream should be true")
	}
}

func TestResponsesStreamChunk_Parse(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantType string
		wantText string
	}{
		{
			name:     "delta",
			json:     `{"type":"response.output_text.delta","delta":"Hello"}`,
			wantType: "response.output_text.delta",
			wantText: "Hello",
		},
		{
			name:     "completed",
			json:     `{"type":"response.completed"}`,
			wantType: "response.completed",
			wantText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var chunk ResponsesStreamChunk
			if err := json.Unmarshal([]byte(tt.json), &chunk); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}
			if chunk.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", chunk.Type, tt.wantType)
			}
			if chunk.Delta != tt.wantText {
				t.Errorf("Delta = %q, want %q", chunk.Delta, tt.wantText)
			}
		})
	}
}

func TestLevelToReasoningEffort(t *testing.T) {
	tests := []struct {
		level string
		want  string
	}{
		{"low", "low"},
		{"medium", "medium"},
		{"high", "high"},
		{"xhigh", "high"},     // xhigh maps to high
		{"unknown", "medium"}, // default
		{"", "medium"},        // default
		{"invalid", "medium"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			got := LevelToReasoningEffort(tt.level)
			if got != tt.want {
				t.Errorf("LevelToReasoningEffort(%q) = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}

func TestIsCodexModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"gpt-5.2-codex", true},
		{"gpt-5.1-codex", true},
		{"gpt-5.1-codex-max", true},
		{"gpt-5-codex", true},
		{"GPT-5.2-CODEX", true}, // case insensitive
		{"gpt-5.2-Codex", true}, // mixed case
		{"gpt-4o", false},
		{"gpt-5.2", false},
		{"claude-3-opus", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := isCodexModel(tt.model)
			if got != tt.want {
				t.Errorf("isCodexModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}
