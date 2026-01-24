package groq

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestNew(t *testing.T) {
	apiKey := "test-api-key"
	provider := New(apiKey)

	if provider == nil {
		t.Fatal("New() returned nil")
	}
}

func TestProvider_Name(t *testing.T) {
	provider := New("test-key")

	name := provider.Name()
	if name != "Groq" {
		t.Errorf("Name() = %v, want 'Groq'", name)
	}
}

func TestProvider_SupportsImages(t *testing.T) {
	provider := New("test-key")

	supports := provider.SupportsImages()
	if supports {
		t.Error("SupportsImages() = true, want false (Groq does not support images)")
	}
}

func TestNew_URLOverride(t *testing.T) {
	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)

	t.Run("DefaultURL", func(t *testing.T) {
		os.Unsetenv("GROQ_API_URL")
		p := New("test-key")
		if p.APIURL() != defaultGroqURL {
			t.Errorf("apiURL = %q, want %q", p.APIURL(), defaultGroqURL)
		}
	})

	t.Run("CustomURL", func(t *testing.T) {
		customURL := "https://custom.groq.api.com/v1"
		os.Setenv("GROQ_API_URL", customURL)
		p := New("test-key")
		if p.APIURL() != customURL {
			t.Errorf("apiURL = %q, want %q", p.APIURL(), customURL)
		}
	})
}

// Helper functions for testing

func mockAPIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func streamingHandler(chunks []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		for _, chunk := range chunks {
			_, _ = w.Write([]byte("data: " + chunk + "\n\n"))
			flusher.Flush()
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}
}

func errorHandler(statusCode int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(message))
	}
}

func rateLimitHandler(retryAfter string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", retryAfter)
		w.WriteHeader(429)
		_, _ = w.Write([]byte("Rate limit exceeded"))
	}
}

func TestProvider_ChatWithTools_NonStreaming(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization = %q, want 'Bearer test-key'", r.Header.Get("Authorization"))
		}

		var req api.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if req.Model != "llama3-70b-8192" {
			t.Errorf("Model = %q, want 'llama3-70b-8192'", req.Model)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := api.ChatResponse{
			Choices: []api.Choice{
				{Message: api.Message{Content: "Test response from Groq"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)
	os.Setenv("GROQ_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	result, err := p.ChatWithTools(context.Background(), "System prompt", history, "llama3-70b-8192")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Test response from Groq" {
		t.Errorf("ChatWithTools() = %q, want 'Test response from Groq'", result)
	}
}

func TestProvider_ChatWithTools_Streaming(t *testing.T) {
	chunks := []string{
		`{"choices":[{"delta":{"content":"Hello"}}]}`,
		`{"choices":[{"delta":{"content":" from"}}]}`,
		`{"choices":[{"delta":{"content":" Groq"}}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)
	os.Setenv("GROQ_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "llama3-70b-8192")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Hello from Groq" {
		t.Errorf("ChatWithTools() = %q, want 'Hello from Groq'", result)
	}
}

func TestProvider_ChatWithTools_APIError(t *testing.T) {
	server := mockAPIServer(t, errorHandler(500, "Internal Server Error"))

	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)
	os.Setenv("GROQ_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for 500 status")
	}
}

func TestProvider_ChatWithTools_RateLimit(t *testing.T) {
	server := mockAPIServer(t, rateLimitHandler("60"))

	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)
	os.Setenv("GROQ_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for rate limit")
	}
}

func TestProvider_ChatWithImage(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := api.ChatResponse{
			Choices: []api.Choice{
				{Message: api.Message{Content: "Image ignored"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)
	os.Setenv("GROQ_API_URL", server.URL)

	p := New("test-key")
	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png"}

	// Groqは画像非対応なのでテキストのみ送信
	result, err := p.ChatWithImage(context.Background(), "System", nil, "Describe this", image, "")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "Image ignored" {
		t.Errorf("ChatWithImage() = %q, want 'Image ignored'", result)
	}
}
