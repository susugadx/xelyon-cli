package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

func TestNewGroqProvider(t *testing.T) {
	apiKey := "test-api-key"
	provider := NewGroqProvider(apiKey)

	if provider == nil {
		t.Fatal("NewGroqProvider() returned nil")
	}
}

func TestGroqProvider_Name(t *testing.T) {
	provider := NewGroqProvider("test-key")

	name := provider.Name()
	if name != "Groq" {
		t.Errorf("Name() = %v, want 'Groq'", name)
	}
}

func TestGroqProvider_SupportsImages(t *testing.T) {
	provider := NewGroqProvider("test-key")

	supports := provider.SupportsImages()
	if supports {
		t.Error("SupportsImages() = true, want false (Groq does not support images)")
	}
}

func TestNewGroqProvider_URLOverride(t *testing.T) {
	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)

	t.Run("DefaultURL", func(t *testing.T) {
		os.Unsetenv("GROQ_API_URL")
		p := NewGroqProvider("test-key")
		if p.apiURL != defaultGroqURL {
			t.Errorf("apiURL = %q, want %q", p.apiURL, defaultGroqURL)
		}
	})

	t.Run("CustomURL", func(t *testing.T) {
		customURL := "https://custom.groq.api.com/v1"
		os.Setenv("GROQ_API_URL", customURL)
		p := NewGroqProvider("test-key")
		if p.apiURL != customURL {
			t.Errorf("apiURL = %q, want %q", p.apiURL, customURL)
		}
	})
}

func TestGroqProvider_ChatWithTools_NonStreaming(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequestMethod(t, r, "POST")
		assertJSONContentType(t, r)
		assertRequestHeader(t, r, "Authorization", "Bearer test-key")

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if req.Model != "llama3-70b-8192" {
			t.Errorf("Model = %q, want 'llama3-70b-8192'", req.Model)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := ChatResponse{
			Choices: []Choice{
				{Message: Message{Content: "Test response from Groq"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)
	os.Setenv("GROQ_API_URL", server.URL)

	p := NewGroqProvider("test-key")
	history := []Message{{Role: "user", Content: "Hello"}}

	result, err := p.ChatWithTools(context.Background(), "System prompt", history, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Test response from Groq" {
		t.Errorf("ChatWithTools() = %q, want 'Test response from Groq'", result)
	}
}

func TestGroqProvider_ChatWithTools_Streaming(t *testing.T) {
	chunks := []string{
		`{"choices":[{"delta":{"content":"Hello"}}]}`,
		`{"choices":[{"delta":{"content":" from"}}]}`,
		`{"choices":[{"delta":{"content":" Groq"}}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)
	os.Setenv("GROQ_API_URL", server.URL)

	p := NewGroqProvider("test-key")
	history := []Message{{Role: "user", Content: "Hi"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "llama3-70b-8192")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Hello from Groq" {
		t.Errorf("ChatWithTools() = %q, want 'Hello from Groq'", result)
	}
}

func TestGroqProvider_ChatWithTools_APIError(t *testing.T) {
	server := mockAPIServer(t, errorHandler(500, "Internal Server Error"))

	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)
	os.Setenv("GROQ_API_URL", server.URL)

	p := NewGroqProvider("test-key")
	history := []Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for 500 status")
	}
}

func TestGroqProvider_ChatWithTools_RateLimit(t *testing.T) {
	server := mockAPIServer(t, rateLimitHandler("60"))

	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)
	os.Setenv("GROQ_API_URL", server.URL)

	p := NewGroqProvider("test-key")
	history := []Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for rate limit")
	}
}

func TestGroqProvider_ChatWithImage(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := ChatResponse{
			Choices: []Choice{
				{Message: Message{Content: "Image ignored"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)
	os.Setenv("GROQ_API_URL", server.URL)

	p := NewGroqProvider("test-key")
	image := &ImageData{Base64: "dGVzdA==", MediaType: "image/png"}

	// Groqは画像非対応なのでテキストのみ送信
	result, err := p.ChatWithImage(context.Background(), "System", nil, "Describe this", image, "")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "Image ignored" {
		t.Errorf("ChatWithImage() = %q, want 'Image ignored'", result)
	}
}
