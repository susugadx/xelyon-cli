package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"

	// ツール登録のための blank import
	_ "github.com/susugadx/xelyon-cli/internal/tools/dev"
	_ "github.com/susugadx/xelyon-cli/internal/tools/file"
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

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestClearToolUses_OpenRouter(t *testing.T) {
	cfg := config.DefaultConfig()

	t.Run("ClaudeModel", func(t *testing.T) {
		contextManagement, betaHeaders := buildOpenRouterClaudeContextManagement(
			"anthropic/claude-sonnet-4.6",
			cfg.Compression,
			nil,
		)

		if contextManagement == nil {
			t.Fatal("ContextManagement should be set for Claude models")
		}
		if len(contextManagement.Edits) != 2 {
			t.Fatalf("len(ContextManagement.Edits) = %d, want 2", len(contextManagement.Edits))
		}
		if contextManagement.Edits[0].Type != "clear_tool_uses_20250919" {
			t.Errorf("Edits[0].Type = %q, want clear_tool_uses_20250919", contextManagement.Edits[0].Type)
		}
		if contextManagement.Edits[1].Type != "compact_20260112" {
			t.Errorf("Edits[1].Type = %q, want compact_20260112", contextManagement.Edits[1].Type)
		}
		if !containsString(betaHeaders, "context-management-2025-06-27") {
			t.Errorf("beta headers should include context-management-2025-06-27, got %v", betaHeaders)
		}
		if !containsString(betaHeaders, "compact-2026-01-12") {
			t.Errorf("beta headers should include compact-2026-01-12, got %v", betaHeaders)
		}
	})

	t.Run("Opus47Compaction", func(t *testing.T) {
		contextManagement, betaHeaders := buildOpenRouterClaudeContextManagement(
			"anthropic/claude-opus-4-7",
			cfg.Compression,
			nil,
		)

		if contextManagement == nil {
			t.Fatal("ContextManagement should be set for OpenRouter Opus 4.7")
		}
		if len(contextManagement.Edits) != 2 {
			t.Fatalf("len(ContextManagement.Edits) = %d, want 2", len(contextManagement.Edits))
		}
		if contextManagement.Edits[1].Type != "compact_20260112" {
			t.Fatalf("Edits[1].Type = %q, want compact_20260112", contextManagement.Edits[1].Type)
		}
		if !containsString(betaHeaders, "compact-2026-01-12") {
			t.Fatalf("beta headers should include compact-2026-01-12, got %v", betaHeaders)
		}
	})

	t.Run("ClearOnlyWithoutCompaction", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Compression.ClaudeCompaction = false

		contextManagement, betaHeaders := buildOpenRouterClaudeContextManagement(
			"anthropic/claude-3.5-sonnet",
			cfg.Compression,
			nil,
		)

		if contextManagement == nil {
			t.Fatal("ContextManagement should be set when clear_tool_uses is enabled")
		}
		if len(contextManagement.Edits) != 1 {
			t.Fatalf("len(ContextManagement.Edits) = %d, want 1", len(contextManagement.Edits))
		}
		if contextManagement.Edits[0].Type != "clear_tool_uses_20250919" {
			t.Errorf("Edits[0].Type = %q, want clear_tool_uses_20250919", contextManagement.Edits[0].Type)
		}
		if !containsString(betaHeaders, "context-management-2025-06-27") {
			t.Errorf("beta headers should include context-management-2025-06-27, got %v", betaHeaders)
		}
		if containsString(betaHeaders, "compact-2026-01-12") {
			t.Errorf("beta headers should not include compact-2026-01-12 when compaction is disabled, got %v", betaHeaders)
		}
	})

	t.Run("NonClaudeModel", func(t *testing.T) {
		contextManagement, betaHeaders := buildOpenRouterClaudeContextManagement(
			"openai/gpt-5.2",
			cfg.Compression,
			[]string{"existing-beta"},
		)

		if contextManagement != nil {
			t.Fatal("ContextManagement should be nil for non-Claude models")
		}
		if len(betaHeaders) != 1 || betaHeaders[0] != "existing-beta" {
			t.Errorf("beta headers = %v, want [existing-beta]", betaHeaders)
		}
	})
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

		var req openaicompat.ChatCompletionsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if req.Model != "openai/gpt-4-turbo" {
			t.Errorf("Model = %q, want 'openai/gpt-4-turbo'", req.Model)
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

	result, err := p.ChatWithTools(context.Background(), "System prompt", history, "openai/gpt-4-turbo")
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
	server := mockAPIServer(t, rateLimitHandler("1"))

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

	result, err := p.ChatWithImage(context.Background(), "System", nil, "Describe this", image, "openai/gpt-4-turbo")
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

func TestProvider_ChatWithImage_ClearToolUsesUsesAnthropicSkin(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ClaudeCompaction = false

	var requestPath string
	var requestBody struct {
		Model             string            `json:"model"`
		AnthropicBeta     []string          `json:"anthropic_beta,omitempty"`
		CacheControl      *api.CacheControl `json:"cache_control,omitempty"`
		Messages          []interface{}     `json:"messages"`
		ContextManagement *struct {
			Edits []struct {
				Type string `json:"type"`
			} `json:"edits"`
		} `json:"context_management,omitempty"`
	}

	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
	})

	t.Setenv("OPENROUTER_API_URL", server.URL+"/v1/chat/completions")

	p := New("test-key")
	ctx := config.WithContext(context.Background(), cfg)
	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png"}

	result, err := p.ChatWithImage(ctx, "System", nil, "Describe this", image, "anthropic/claude-3.5-sonnet")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "Hello" {
		t.Errorf("ChatWithImage() = %q, want %q", result, "Hello")
	}
	if requestPath != "/v1/messages" {
		t.Errorf("request path = %q, want %q", requestPath, "/v1/messages")
	}
	if requestBody.Model != "anthropic/claude-3.5-sonnet" {
		t.Errorf("Model = %q, want %q", requestBody.Model, "anthropic/claude-3.5-sonnet")
	}
	if requestBody.ContextManagement == nil || len(requestBody.ContextManagement.Edits) != 1 {
		t.Fatal("ContextManagement should contain only clear_tool_uses")
	}
	if requestBody.ContextManagement.Edits[0].Type != "clear_tool_uses_20250919" {
		t.Errorf("Edits[0].Type = %q, want clear_tool_uses_20250919", requestBody.ContextManagement.Edits[0].Type)
	}
	if requestBody.CacheControl == nil || requestBody.CacheControl.Type != "ephemeral" {
		t.Fatalf("cache_control = %+v, want top-level ephemeral cache_control", requestBody.CacheControl)
	}
	if !containsString(requestBody.AnthropicBeta, "context-management-2025-06-27") {
		t.Errorf("anthropic_beta should include context-management-2025-06-27, got %v", requestBody.AnthropicBeta)
	}
	if containsString(requestBody.AnthropicBeta, "compact-2026-01-12") {
		t.Errorf("anthropic_beta should not include compact-2026-01-12, got %v", requestBody.AnthropicBeta)
	}
	if len(requestBody.Messages) == 0 {
		t.Fatal("Messages should not be empty")
	}

	lastMsg, ok := requestBody.Messages[len(requestBody.Messages)-1].(map[string]interface{})
	if !ok {
		t.Fatalf("Last message should be a map, got %T", requestBody.Messages[len(requestBody.Messages)-1])
	}
	content, ok := lastMsg["content"].([]interface{})
	if !ok {
		t.Fatalf("Content should be an array for multimodal message, got %T", lastMsg["content"])
	}
	if len(content) != 2 {
		t.Fatalf("Content length = %d, want 2 (image + text)", len(content))
	}

	firstPart, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatalf("First content part should be a map, got %T", content[0])
	}
	if firstPart["type"] != "image" {
		t.Errorf("First content type = %v, want %q", firstPart["type"], "image")
	}

	secondPart, ok := content[1].(map[string]interface{})
	if !ok {
		t.Fatalf("Second content part should be a map, got %T", content[1])
	}
	if secondPart["type"] != "text" {
		t.Errorf("Second content type = %v, want %q", secondPart["type"], "text")
	}
	if secondPart["text"] != "Describe this" {
		t.Errorf("Second content text = %v, want %q", secondPart["text"], "Describe this")
	}
}

func TestProvider_ChatWithTools_ClearToolUsesUsesAnthropicSkin(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ClaudeCompaction = false

	var requestPath string
	var requestBody struct {
		Model             string            `json:"model"`
		AnthropicBeta     []string          `json:"anthropic_beta,omitempty"`
		CacheControl      *api.CacheControl `json:"cache_control,omitempty"`
		ContextManagement *struct {
			Edits []struct {
				Type string `json:"type"`
			} `json:"edits"`
		} `json:"context_management,omitempty"`
	}

	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
	})

	t.Setenv("OPENROUTER_API_URL", server.URL+"/v1/chat/completions")

	p := New("test-key")
	ctx := config.WithContext(context.Background(), cfg)
	result, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "Hello"}}, "anthropic/claude-3.5-sonnet")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Hello" {
		t.Errorf("ChatWithTools() = %q, want %q", result, "Hello")
	}
	if requestPath != "/v1/messages" {
		t.Errorf("request path = %q, want %q", requestPath, "/v1/messages")
	}
	if requestBody.ContextManagement == nil || len(requestBody.ContextManagement.Edits) != 1 {
		t.Fatal("ContextManagement should contain only clear_tool_uses")
	}
	if requestBody.ContextManagement.Edits[0].Type != "clear_tool_uses_20250919" {
		t.Errorf("Edits[0].Type = %q, want clear_tool_uses_20250919", requestBody.ContextManagement.Edits[0].Type)
	}
	if requestBody.CacheControl == nil || requestBody.CacheControl.Type != "ephemeral" {
		t.Fatalf("cache_control = %+v, want top-level ephemeral cache_control", requestBody.CacheControl)
	}
	if !containsString(requestBody.AnthropicBeta, "context-management-2025-06-27") {
		t.Errorf("anthropic_beta should include context-management-2025-06-27, got %v", requestBody.AnthropicBeta)
	}
	if containsString(requestBody.AnthropicBeta, "compact-2026-01-12") {
		t.Errorf("anthropic_beta should not include compact-2026-01-12, got %v", requestBody.AnthropicBeta)
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

	var requestBody openaicompat.ChatCompletionsRequest
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

	var requestBody openaicompat.ChatCompletionsRequest
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
		{"Anthropic/Claude-Opus-4.7", true},
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
		{"anthropic/claude-opus-4-7", true},
		{"anthropic/claude-opus-4.7", true},
		{"anthropic/claude-opus-4-6", true},
		{"anthropic/claude-sonnet-4.6", true},
		{"anthropic/claude-sonnet-4-6", true},
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

	spinner := api.StartThinkingSpinner(context.Background(), false, "")
	result, err := p.handleClaudeStreamingResponse(context.Background(), resp, spinner)
	if err != nil {
		t.Fatalf("handleClaudeStreamingResponse() error = %v", err)
	}

	want := "Hello World"
	if result != want {
		t.Errorf("handleClaudeStreamingResponse() = %q, want %q", result, want)
	}
}

func TestStreamingResponses_DoNotEmitBlankLineWhenAssistantUpdatesSuppressed(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		wantText bool
	}{
		{name: "verbose", mode: api.AssistantUpdatesVerbose, wantText: true},
		{name: "phase", mode: api.AssistantUpdatesPhase, wantText: false},
		{name: "off", mode: api.AssistantUpdatesOff, wantText: false},
	}

	for _, tt := range tests {
		t.Run("openai-"+tt.name, func(t *testing.T) {
			chunks := []string{
				`{"choices":[{"delta":{"content":"Hello"}}]}`,
				`{"choices":[{"delta":{"content":" from"}}]}`,
				`{"choices":[{"delta":{"content":" OpenRouter"}}]}`,
			}
			server := mockAPIServer(t, streamingHandler(chunks))
			defer server.Close()

			resp, err := http.Post(server.URL, "application/json", nil)
			if err != nil {
				t.Fatalf("Failed to post to mock server: %v", err)
			}
			defer resp.Body.Close()

			var out bytes.Buffer
			ctx := ui.WithRuntime(context.Background(), ui.NewRuntime(nil, &out, &out))
			ctx = api.WithAssistantUpdateMode(ctx, tt.mode)

			p := New("test-key")
			result, err := p.handleStreamingResponse(ctx, resp, ui.NewSpinnerWithWriter(io.Discard))
			if err != nil {
				t.Fatalf("handleStreamingResponse() error = %v", err)
			}
			if result != "Hello from OpenRouter" {
				t.Fatalf("handleStreamingResponse() = %q, want %q", result, "Hello from OpenRouter")
			}

			output := out.String()
			if tt.wantText {
				if !strings.Contains(output, "Hello from OpenRouter") {
					t.Fatalf("expected streamed assistant text in verbose mode, got: %q", output)
				}
				if !strings.HasSuffix(output, "\n") {
					t.Fatalf("expected trailing newline in verbose mode, got: %q", output)
				}
			} else if output != "" {
				t.Fatalf("expected no streamed prose or stray newline in %s mode, got: %q", tt.mode, output)
			}
		})

		t.Run("claude-"+tt.name, func(t *testing.T) {
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

			resp, err := http.Post(server.URL, "application/json", nil)
			if err != nil {
				t.Fatalf("Failed to post to mock server: %v", err)
			}
			defer resp.Body.Close()

			var out bytes.Buffer
			ctx := ui.WithRuntime(context.Background(), ui.NewRuntime(nil, &out, &out))
			ctx = api.WithAssistantUpdateMode(ctx, tt.mode)

			p := New("test-key")
			result, err := p.handleClaudeStreamingResponse(ctx, resp, ui.NewSpinnerWithWriter(io.Discard))
			if err != nil {
				t.Fatalf("handleClaudeStreamingResponse() error = %v", err)
			}
			if result != "Hello World" {
				t.Fatalf("handleClaudeStreamingResponse() = %q, want %q", result, "Hello World")
			}

			output := out.String()
			if tt.wantText {
				if !strings.Contains(output, "Hello World") {
					t.Fatalf("expected streamed assistant text in verbose mode, got: %q", output)
				}
				if !strings.HasSuffix(output, "\n") {
					t.Fatalf("expected trailing newline in verbose mode, got: %q", output)
				}
			} else if output != "" {
				t.Fatalf("expected no streamed prose or stray newline in %s mode, got: %q", tt.mode, output)
			}
		})
	}
}
