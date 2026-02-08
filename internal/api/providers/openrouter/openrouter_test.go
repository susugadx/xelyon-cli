package openrouter

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
	if name != "OpenRouter" {
		t.Errorf("Name() = %v, want 'OpenRouter'", name)
	}
}

func TestProvider_SupportsImages(t *testing.T) {
	provider := New("test-key")

	supports := provider.SupportsImages()
	if !supports {
		t.Error("SupportsImages() = false, want true (OpenRouter supports images for some models)")
	}
}

func TestNew_URLOverride(t *testing.T) {
	originalURL := os.Getenv("OPENROUTER_API_URL")
	defer os.Setenv("OPENROUTER_API_URL", originalURL)

	t.Run("DefaultURL", func(t *testing.T) {
		os.Unsetenv("OPENROUTER_API_URL")
		p := New("test-key")
		if p.APIURL != defaultOpenRouterURL {
			t.Errorf("apiURL = %q, want %q", p.APIURL, defaultOpenRouterURL)
		}
	})

	t.Run("CustomURL", func(t *testing.T) {
		customURL := "https://custom.openrouter.api.com/v1"
		os.Setenv("OPENROUTER_API_URL", customURL)
		p := New("test-key")
		if p.APIURL != customURL {
			t.Errorf("apiURL = %q, want %q", p.APIURL, customURL)
		}
	})
}

func TestProvider_IsFunctionCallingEnabled(t *testing.T) {
	originalEnv := os.Getenv("OPENROUTER_FUNCTION_CALLING")
	defer os.Setenv("OPENROUTER_FUNCTION_CALLING", originalEnv)

	t.Run("EnabledByDefault", func(t *testing.T) {
		os.Unsetenv("OPENROUTER_FUNCTION_CALLING")
		p := New("test-key")
		if !p.IsFunctionCallingEnabled() {
			t.Error("IsFunctionCallingEnabled() = false, want true by default")
		}
	})

	t.Run("DisabledWithEnvVar", func(t *testing.T) {
		os.Setenv("OPENROUTER_FUNCTION_CALLING", "0")
		p := New("test-key")
		if p.IsFunctionCallingEnabled() {
			t.Error("IsFunctionCallingEnabled() = true, want false when OPENROUTER_FUNCTION_CALLING=0")
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
		// OpenRouter固有ヘッダーの確認
		if r.Header.Get("HTTP-Referer") == "" {
			t.Error("HTTP-Referer header should be set")
		}
		if r.Header.Get("X-Title") != "XELYON CLI" {
			t.Errorf("X-Title = %q, want 'XELYON CLI'", r.Header.Get("X-Title"))
		}

		var req api.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if req.Model != "anthropic/claude-3.5-sonnet" {
			t.Errorf("Model = %q, want 'anthropic/claude-3.5-sonnet'", req.Model)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := api.ChatResponse{
			Choices: []api.Choice{
				{Message: api.Message{Content: "Test response from OpenRouter"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("OPENROUTER_API_URL")
	defer os.Setenv("OPENROUTER_API_URL", originalURL)
	os.Setenv("OPENROUTER_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	result, err := p.ChatWithTools(context.Background(), "System prompt", history, "anthropic/claude-3.5-sonnet")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Test response from OpenRouter" {
		t.Errorf("ChatWithTools() = %q, want 'Test response from OpenRouter'", result)
	}
}

func TestProvider_ChatWithTools_Streaming(t *testing.T) {
	chunks := []string{
		`{"choices":[{"delta":{"content":"Hello"}}]}`,
		`{"choices":[{"delta":{"content":" from"}}]}`,
		`{"choices":[{"delta":{"content":" OpenRouter"}}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("OPENROUTER_API_URL")
	defer os.Setenv("OPENROUTER_API_URL", originalURL)
	os.Setenv("OPENROUTER_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "openai/gpt-4-turbo")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Hello from OpenRouter" {
		t.Errorf("ChatWithTools() = %q, want 'Hello from OpenRouter'", result)
	}
}

func TestProvider_ChatWithTools_APIError(t *testing.T) {
	server := mockAPIServer(t, errorHandler(500, "Internal Server Error"))

	originalURL := os.Getenv("OPENROUTER_API_URL")
	defer os.Setenv("OPENROUTER_API_URL", originalURL)
	os.Setenv("OPENROUTER_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for 500 status")
	}
}

func TestProvider_ChatWithTools_RateLimit(t *testing.T) {
	server := mockAPIServer(t, rateLimitHandler("60"))

	originalURL := os.Getenv("OPENROUTER_API_URL")
	defer os.Setenv("OPENROUTER_API_URL", originalURL)
	os.Setenv("OPENROUTER_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for rate limit")
	}
}

func TestProvider_ChatWithImage(t *testing.T) {
	var requestBody map[string]interface{}
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := api.ChatResponse{
			Choices: []api.Choice{
				{Message: api.Message{Content: "I can see the image"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("OPENROUTER_API_URL")
	defer os.Setenv("OPENROUTER_API_URL", originalURL)
	os.Setenv("OPENROUTER_API_URL", server.URL)

	p := New("test-key")
	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png"}

	result, err := p.ChatWithImage(context.Background(), "System", nil, "Describe this", image, "anthropic/claude-3.5-sonnet")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "I can see the image" {
		t.Errorf("ChatWithImage() = %q, want 'I can see the image'", result)
	}

	// リクエストボディにメッセージが含まれていることを確認
	messages, ok := requestBody["messages"].([]interface{})
	if !ok || len(messages) == 0 {
		t.Fatal("Messages should not be empty")
	}

	// 最後のメッセージが画像付きであることを確認
	lastMsg, ok := messages[len(messages)-1].(map[string]interface{})
	if !ok {
		t.Fatal("Last message should be a map")
	}
	content, ok := lastMsg["content"].([]interface{})
	if !ok {
		t.Fatalf("Content should be an array for multimodal message, got %T", lastMsg["content"])
	}
	if len(content) != 2 {
		t.Errorf("Content length = %d, want 2 (image + text)", len(content))
	}
}

// Function Calling Tests

func TestProvider_ChatWithTools_ToolCalls(t *testing.T) {
	originalEnv := os.Getenv("OPENROUTER_FUNCTION_CALLING")
	defer os.Setenv("OPENROUTER_FUNCTION_CALLING", originalEnv)
	os.Unsetenv("OPENROUTER_FUNCTION_CALLING")

	chunks := []string{
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_123","type":"function","function":{"name":"read_file","arguments":""}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"pa"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"th\":\"/te"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"st.txt\"}"}}]}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("OPENROUTER_API_URL")
	defer os.Setenv("OPENROUTER_API_URL", originalURL)
	os.Setenv("OPENROUTER_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Read test.txt"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "openai/gpt-4-turbo")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	if result == "" {
		t.Error("ChatWithTools() returned empty result, expected tool JSON")
	}
	if !contains(result, "read_file") {
		t.Errorf("ChatWithTools() = %q, expected to contain 'read_file'", result)
	}
	if !contains(result, "call_123") {
		t.Errorf("ChatWithTools() = %q, expected to contain 'call_123'", result)
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

func TestSetUsageCallback(t *testing.T) {
	p := New("test-key")

	var calledWith api.Usage
	callback := func(u api.Usage) {
		calledWith = u
	}
	p.SetUsageCallback(callback)

	// コールバックが設定されていることを確認
	if p.usageCallback == nil {
		t.Error("usageCallback should not be nil")
	}

	// コールバックが正しく呼ばれることを確認
	p.usageCallback(api.Usage{InputTokens: 100, OutputTokens: 50})
	if calledWith.InputTokens != 100 || calledWith.OutputTokens != 50 {
		t.Errorf("Callback received wrong values: %+v", calledWith)
	}
}

func TestProvider_ChatWithTools_FunctionCallingDisabled(t *testing.T) {
	originalEnv := os.Getenv("OPENROUTER_FUNCTION_CALLING")
	defer os.Setenv("OPENROUTER_FUNCTION_CALLING", originalEnv)
	os.Setenv("OPENROUTER_FUNCTION_CALLING", "0")

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

	originalURL := os.Getenv("OPENROUTER_API_URL")
	defer os.Setenv("OPENROUTER_API_URL", originalURL)
	os.Setenv("OPENROUTER_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "openai/gpt-4-turbo")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	if len(requestBody.Tools) > 0 {
		t.Errorf("Tools should be empty when OPENROUTER_FUNCTION_CALLING=0, got %d tools", len(requestBody.Tools))
	}
}

func TestProvider_ChatWithTools_FunctionCallingEnabled(t *testing.T) {
	originalEnv := os.Getenv("OPENROUTER_FUNCTION_CALLING")
	defer os.Setenv("OPENROUTER_FUNCTION_CALLING", originalEnv)
	os.Unsetenv("OPENROUTER_FUNCTION_CALLING")

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

	originalURL := os.Getenv("OPENROUTER_API_URL")
	defer os.Setenv("OPENROUTER_API_URL", originalURL)
	os.Setenv("OPENROUTER_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "openai/gpt-4-turbo")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	if len(requestBody.Tools) == 0 {
		t.Error("Tools should not be empty when OPENROUTER_FUNCTION_CALLING is not disabled")
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

func TestIsClaudeModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"anthropic/claude-opus-4.5", true},
		{"anthropic/claude-3-5-sonnet", true},
		{"google/gemini-pro", false},
		{"openai/gpt-4o", false},
	}

	for _, tt := range tests {
		if got := isClaudeModel(tt.model); got != tt.want {
			t.Errorf("isClaudeModel(%q) = %v, want %v", tt.model, got, tt.want)
		}
	}
}

func TestIsCompactionSupported(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"anthropic/claude-opus-4.5", true},
		{"anthropic/claude-opus-4-6", true},
		{"anthropic/claude-3-5-sonnet", false},
		{"openai/gpt-4o", false},
	}

	for _, tt := range tests {
		if got := isCompactionSupported(tt.model); got != tt.want {
			t.Errorf("isCompactionSupported(%q) = %v, want %v", tt.model, got, tt.want)
		}
	}
}

func TestGetAnthropicSkinURL(t *testing.T) {
	tests := []struct {
		openaiURL string
		want      string
	}{
		{"https://openrouter.ai/api/v1/chat/completions", "https://openrouter.ai/api/v1/messages"},
		{"https://example.com/v1/chat/completions", "https://example.com/v1/messages"},
		{"https://api.com/chat/completions", "https://api.com/messages"},
	}

	for _, tt := range tests {
		if got := getAnthropicSkinURL(tt.openaiURL); got != tt.want {
			t.Errorf("getAnthropicSkinURL(%q) = %q, want %q", tt.openaiURL, got, tt.want)
		}
	}
}

func TestHandleClaudeStreamingResponse(t *testing.T) {
	chunks := []string{
		`{"type": "message_start", "message": {"usage": {"input_tokens": 10}}}`,
		`{"type": "content_block_start", "index": 0, "content_block": {"type": "text", "text": ""}}`,
		`{"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": "Hello"}}`,
		`{"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": " World"}}`,
		`{"type": "content_block_stop", "index": 0}`,
		`{"type": "message_delta", "delta": {"stop_reason": "end_turn"}, "usage": {"output_tokens": 5}}`,
		`{"type": "message_stop"}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))
	defer server.Close()

	p := New("test-key")
	resp, err := http.Post(server.URL, "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to post to mock server: %v", err)
	}
	defer resp.Body.Close()

	spinner := api.StartThinkingSpinner(false, "")
	result, err := p.handleClaudeStreamingResponse(context.Background(), resp, spinner)
	if err != nil {
		t.Fatalf("handleClaudeStreamingResponse() error = %v", err)
	}

	want := "Hello World"
	if result != want {
		t.Errorf("handleClaudeStreamingResponse() = %q, want %q", result, want)
	}
}
