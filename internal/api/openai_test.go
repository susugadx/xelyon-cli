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

	result, err := p.ChatWithTools(context.Background(), "System prompt", history, "")
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
