package groq

import (
	"context"
	"encoding/json"
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

// Function Calling Tests

func TestProvider_ChatWithTools_ToolCalls(t *testing.T) {
	// Disable function calling env var for this test setup
	originalEnv := os.Getenv("GROQ_FUNCTION_CALLING")
	defer os.Setenv("GROQ_FUNCTION_CALLING", originalEnv)
	os.Unsetenv("GROQ_FUNCTION_CALLING")

	chunks := []string{
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_123","type":"function","function":{"name":"read_file","arguments":""}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"pa"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"th\":\"/te"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"st.txt\"}"}}]}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)
	os.Setenv("GROQ_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Read test.txt"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "llama3-70b-8192")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// Should contain tool JSON
	if result == "" {
		t.Error("ChatWithTools() returned empty result, expected tool JSON")
	}
	// Check for expected tool call format
	if !contains(result, "read_file") {
		t.Errorf("ChatWithTools() = %q, expected to contain 'read_file'", result)
	}
	if !contains(result, "call_123") {
		t.Errorf("ChatWithTools() = %q, expected to contain 'call_123'", result)
	}
}

func TestProvider_ChatWithTools_MultipleToolCalls(t *testing.T) {
	originalEnv := os.Getenv("GROQ_FUNCTION_CALLING")
	defer os.Setenv("GROQ_FUNCTION_CALLING", originalEnv)
	os.Unsetenv("GROQ_FUNCTION_CALLING")

	chunks := []string{
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":""}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"path\":\"/a.txt\"}"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":1,"id":"call_2","type":"function","function":{"name":"read_file","arguments":""}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":1,"function":{"arguments":"{\"path\":\"/b.txt\"}"}}]}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)
	os.Setenv("GROQ_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Read both files"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "llama3-70b-8192")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// Should contain both tool calls
	if !contains(result, "call_1") {
		t.Errorf("ChatWithTools() = %q, expected to contain 'call_1'", result)
	}
	if !contains(result, "call_2") {
		t.Errorf("ChatWithTools() = %q, expected to contain 'call_2'", result)
	}
}

func TestProvider_ChatWithTools_TextWithToolCalls(t *testing.T) {
	originalEnv := os.Getenv("GROQ_FUNCTION_CALLING")
	defer os.Setenv("GROQ_FUNCTION_CALLING", originalEnv)
	os.Unsetenv("GROQ_FUNCTION_CALLING")

	chunks := []string{
		`{"choices":[{"delta":{"content":"Let me read that file"}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_abc","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"/test.txt\"}"}}]}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)
	os.Setenv("GROQ_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Read test.txt"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "llama3-70b-8192")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// Should contain both text and tool call
	if !contains(result, "Let me read that file") {
		t.Errorf("ChatWithTools() = %q, expected to contain text", result)
	}
	if !contains(result, "read_file") {
		t.Errorf("ChatWithTools() = %q, expected to contain 'read_file'", result)
	}
}

func TestSetMCPTools(t *testing.T) {
	p := New("test-key")

	tools := []api.ToolDefinition{
		{Name: "custom_tool", Description: "A custom tool"},
	}
	p.SetMCPTools(tools)

	if len(p.mcpTools) != 1 {
		t.Errorf("mcpTools length = %d, want 1", len(p.mcpTools))
	}
	if p.mcpTools[0].Name != "custom_tool" {
		t.Errorf("mcpTools[0].Name = %q, want 'custom_tool'", p.mcpTools[0].Name)
	}
}

func TestProvider_ChatWithTools_FunctionCallingDisabled(t *testing.T) {
	originalEnv := os.Getenv("GROQ_FUNCTION_CALLING")
	defer os.Setenv("GROQ_FUNCTION_CALLING", originalEnv)
	os.Setenv("GROQ_FUNCTION_CALLING", "0")

	var requestBody api.ChatRequest
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := api.ChatResponse{
			Choices: []api.Choice{
				{Message: api.Message{Content: "No tools"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)
	os.Setenv("GROQ_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "llama3-70b-8192")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// When GROQ_FUNCTION_CALLING=0, Tools should not be included
	if len(requestBody.Tools) > 0 {
		t.Errorf("Tools should be empty when GROQ_FUNCTION_CALLING=0, got %d tools", len(requestBody.Tools))
	}
}

func TestProvider_ChatWithTools_FunctionCallingEnabled(t *testing.T) {
	originalEnv := os.Getenv("GROQ_FUNCTION_CALLING")
	defer os.Setenv("GROQ_FUNCTION_CALLING", originalEnv)
	os.Unsetenv("GROQ_FUNCTION_CALLING")

	var requestBody api.ChatRequest
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := api.ChatResponse{
			Choices: []api.Choice{
				{Message: api.Message{Content: "With tools"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("GROQ_API_URL")
	defer os.Setenv("GROQ_API_URL", originalURL)
	os.Setenv("GROQ_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "llama3-70b-8192")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// When GROQ_FUNCTION_CALLING is not "0", Tools should be included
	if len(requestBody.Tools) == 0 {
		t.Error("Tools should not be empty when GROQ_FUNCTION_CALLING is not disabled")
	}
	if requestBody.ToolChoice != "auto" {
		t.Errorf("ToolChoice = %q, want 'auto'", requestBody.ToolChoice)
	}
}

// Helper function for string contains
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
