package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

func TestNewOpenAIProvider(t *testing.T) {
	apiKey := "test-api-key"
	provider := NewOpenAIProvider(apiKey)

	if provider == nil {
		t.Fatal("NewOpenAIProvider() returned nil")
	}
}

func TestOpenAIProvider_Name(t *testing.T) {
	provider := NewOpenAIProvider("test-key")

	name := provider.Name()
	if name != "OpenAI" {
		t.Errorf("Name() = %v, want 'OpenAI'", name)
	}
}

func TestOpenAIProvider_SupportsImages(t *testing.T) {
	provider := NewOpenAIProvider("test-key")

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
		p := NewOpenAIProvider("test-key")
		if p.apiURL != defaultOpenAIURL {
			t.Errorf("apiURL = %q, want %q", p.apiURL, defaultOpenAIURL)
		}
	})

	t.Run("CustomURL", func(t *testing.T) {
		customURL := "https://custom.openai.api.com/v1"
		os.Setenv("OPENAI_API_URL", customURL)
		p := NewOpenAIProvider("test-key")
		if p.apiURL != customURL {
			t.Errorf("apiURL = %q, want %q", p.apiURL, customURL)
		}
	})
}

func TestOpenAIProvider_ChatWithTools_NonStreaming(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequestMethod(t, r, "POST")
		assertJSONContentType(t, r)
		assertRequestHeader(t, r, "Authorization", "Bearer test-key")

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if req.Model != "gpt-4o" {
			t.Errorf("Model = %q, want 'gpt-4o'", req.Model)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := ChatResponse{
			Choices: []Choice{
				{Message: Message{Content: "Test response from OpenAI"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("OPENAI_API_URL")
	defer os.Setenv("OPENAI_API_URL", originalURL)
	os.Setenv("OPENAI_API_URL", server.URL)

	p := NewOpenAIProvider("test-key")
	history := []Message{{Role: "user", Content: "Hello"}}

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

	p := NewOpenAIProvider("test-key")
	history := []Message{{Role: "user", Content: "Hi"}}

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

	p := NewOpenAIProvider("test-key")
	history := []Message{{Role: "user", Content: "Hi"}}

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

	p := NewOpenAIProvider("test-key")
	history := []Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for rate limit")
	}
}

func TestOpenAIProvider_ChatWithImage_NoImage(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := ChatResponse{
			Choices: []Choice{
				{Message: Message{Content: "No image response"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("OPENAI_API_URL")
	defer os.Setenv("OPENAI_API_URL", originalURL)
	os.Setenv("OPENAI_API_URL", server.URL)

	p := NewOpenAIProvider("test-key")

	result, err := p.ChatWithImage(context.Background(), "System", nil, "Hello", nil, "")
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
		var req OpenAIMultimodalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if len(req.Messages) < 2 { // system + multimodal message
			t.Errorf("Messages count = %d, want at least 2", len(req.Messages))
		}

		w.Header().Set("Content-Type", "application/json")
		resp := ChatResponse{
			Choices: []Choice{
				{Message: Message{Content: "Image analysis complete"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("OPENAI_API_URL")
	defer os.Setenv("OPENAI_API_URL", originalURL)
	os.Setenv("OPENAI_API_URL", server.URL)

	p := NewOpenAIProvider("test-key")
	image := &ImageData{Base64: "dGVzdA==", MediaType: "image/png"}

	result, err := p.ChatWithImage(context.Background(), "System", nil, "Describe this", image, "")
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

	p := NewOpenAIProvider("test-key")
	history := []Message{{Role: "user", Content: "Hi"}}

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
		assertRequestMethod(t, r, "POST")
		assertJSONContentType(t, r)

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

	p := NewOpenAIProvider("test-key")
	history := []Message{
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
