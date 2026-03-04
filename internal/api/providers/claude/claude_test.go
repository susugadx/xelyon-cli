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

	// ツール登録のための blank import
	_ "github.com/susugadx/xelyon-cli/internal/tools/dev"
	_ "github.com/susugadx/xelyon-cli/internal/tools/file"
	_ "github.com/susugadx/xelyon-cli/internal/tools/lsp"
	_ "github.com/susugadx/xelyon-cli/internal/tools/search"
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
		if p.APIURL != defaultClaudeURL {
			t.Errorf("apiURL = %q, want %q", p.APIURL, defaultClaudeURL)
		}
	})

	t.Run("CustomURL", func(t *testing.T) {
		customURL := "https://custom.anthropic.api.com/v1"
		os.Setenv("ANTHROPIC_API_URL", customURL)
		p := New("test-key")
		if p.APIURL != customURL {
			t.Errorf("apiURL = %q, want %q", p.APIURL, customURL)
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
				Delta: &Delta{Type: "text_delta", Text: text},
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

// claudeToolUseStreamingHandler は Tool Use を含むストリーミングハンドラー
func claudeToolUseStreamingHandler(toolID, toolName string, inputChunks []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// content_block_start (tool_use)
		startEvent := StreamEvent{
			Type:  "content_block_start",
			Index: 0,
			ContentBlock: &ContentBlock{
				Type: "tool_use",
				ID:   toolID,
				Name: toolName,
			},
		}
		data, _ := json.Marshal(startEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		// content_block_delta (input_json_delta)
		for _, chunk := range inputChunks {
			deltaEvent := StreamEvent{
				Type:  "content_block_delta",
				Index: 0,
				Delta: &Delta{Type: "input_json_delta", PartialJSON: chunk},
			}
			data, _ := json.Marshal(deltaEvent)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}

		// content_block_stop
		stopBlockEvent := StreamEvent{
			Type:  "content_block_stop",
			Index: 0,
		}
		data, _ = json.Marshal(stopBlockEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		// message_delta (stop_reason)
		msgDeltaEvent := StreamEvent{
			Type:  "message_delta",
			Delta: &Delta{StopReason: "tool_use"},
		}
		data, _ = json.Marshal(msgDeltaEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		// message_stop
		stopEvent := StreamEvent{Type: "message_stop"}
		data, _ = json.Marshal(stopEvent)
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
		if req.Model != "claude-sonnet-4-6" {
			t.Errorf("Model = %q, want 'claude-sonnet-4-6'", req.Model)
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

	result, err := p.ChatWithTools(context.Background(), "System prompt", history, "claude-sonnet-4-6")
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

	result, err := p.ChatWithTools(context.Background(), "System", history, "claude-sonnet-4-6")
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
	server := mockAPIServer(t, rateLimitHandler("1"))

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

// Tool Use Tests

func TestClaudeProvider_ChatWithTools_ToolUse(t *testing.T) {
	// Disable function calling env var for consistent testing
	originalEnv := os.Getenv("CLAUDE_FUNCTION_CALLING")
	defer os.Setenv("CLAUDE_FUNCTION_CALLING", originalEnv)
	os.Unsetenv("CLAUDE_FUNCTION_CALLING")

	inputChunks := []string{
		`{"pa`,
		`th":"`,
		`/test.txt"`,
		`}`,
	}
	server := mockAPIServer(t, claudeToolUseStreamingHandler("toolu_01ABC123", "read_file", inputChunks))

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Read test.txt"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// Should contain tool JSON
	if result == "" {
		t.Error("ChatWithTools() returned empty result, expected tool JSON")
	}
	if !contains(result, "read_file") {
		t.Errorf("ChatWithTools() = %q, expected to contain 'read_file'", result)
	}
	if !contains(result, "toolu_01ABC123") {
		t.Errorf("ChatWithTools() = %q, expected to contain 'toolu_01ABC123'", result)
	}
}

func TestClaudeProvider_ChatWithTools_NonStreaming_ToolUse(t *testing.T) {
	originalEnv := os.Getenv("CLAUDE_FUNCTION_CALLING")
	defer os.Setenv("CLAUDE_FUNCTION_CALLING", originalEnv)
	os.Unsetenv("CLAUDE_FUNCTION_CALLING")

	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := Response{
			Content: []Content{
				{Type: "text", Text: "I'll read that file."},
				{
					Type:  "tool_use",
					ID:    "toolu_01XYZ789",
					Name:  "read_file",
					Input: map[string]interface{}{"path": "/readme.md"},
				},
			},
			StopReason: "tool_use",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Read readme.md"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	if !contains(result, "I'll read that file.") {
		t.Errorf("ChatWithTools() = %q, expected to contain text", result)
	}
	if !contains(result, "read_file") {
		t.Errorf("ChatWithTools() = %q, expected to contain 'read_file'", result)
	}
	if !contains(result, "toolu_01XYZ789") {
		t.Errorf("ChatWithTools() = %q, expected to contain 'toolu_01XYZ789'", result)
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

func TestClaudeProvider_ChatWithTools_FunctionCallingDisabled(t *testing.T) {
	originalEnv := os.Getenv("CLAUDE_FUNCTION_CALLING")
	defer os.Setenv("CLAUDE_FUNCTION_CALLING", originalEnv)
	os.Setenv("CLAUDE_FUNCTION_CALLING", "0")

	var requestBody Request
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := Response{
			Content: []Content{{Type: "text", Text: "No tools"}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// When CLAUDE_FUNCTION_CALLING=0, Tools should not be included
	if len(requestBody.Tools) > 0 {
		t.Errorf("Tools should be empty when CLAUDE_FUNCTION_CALLING=0, got %d tools", len(requestBody.Tools))
	}
}

func TestClaudeProvider_ChatWithTools_FunctionCallingEnabled(t *testing.T) {
	originalEnv := os.Getenv("CLAUDE_FUNCTION_CALLING")
	defer os.Setenv("CLAUDE_FUNCTION_CALLING", originalEnv)
	os.Unsetenv("CLAUDE_FUNCTION_CALLING")

	var requestBody Request
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := Response{
			Content: []Content{{Type: "text", Text: "With tools"}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// When CLAUDE_FUNCTION_CALLING is not "0", Tools should be included
	if len(requestBody.Tools) == 0 {
		t.Error("Tools should not be empty when CLAUDE_FUNCTION_CALLING is not disabled")
	}
}

func TestGetClaudeToolDefinitions(t *testing.T) {
	tools := GetClaudeToolDefinitions()

	if len(tools) == 0 {
		t.Error("GetClaudeToolDefinitions() returned empty slice")
	}

	// read_file ツールが含まれていることを確認
	found := false
	for _, tool := range tools {
		if tool.Name == "read_file" {
			found = true
			if tool.InputSchema == nil {
				t.Error("read_file tool should have InputSchema")
			}
			break
		}
	}
	if !found {
		t.Error("GetClaudeToolDefinitions() should contain 'read_file' tool")
	}
}

func TestConvertOpenAIToolToClaude(t *testing.T) {
	openaiTool := api.ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"arg1": map[string]interface{}{"type": "string"},
			},
		},
	}

	claudeTool := ConvertOpenAIToolToClaude(openaiTool)

	if claudeTool.Name != "test_tool" {
		t.Errorf("Name = %q, want 'test_tool'", claudeTool.Name)
	}
	if claudeTool.Description != "A test tool" {
		t.Errorf("Description = %q, want 'A test tool'", claudeTool.Description)
	}
	if claudeTool.InputSchema == nil {
		t.Error("InputSchema should not be nil")
	}
}

func TestConvertToolUseToToolJSON(t *testing.T) {
	input := map[string]interface{}{
		"path": "/test.txt",
	}

	result, err := ConvertToolUseToToolJSON("toolu_01ABC", "read_file", input)
	if err != nil {
		t.Fatalf("ConvertToolUseToToolJSON() error = %v", err)
	}

	if !contains(result, "toolu_01ABC") {
		t.Errorf("result = %q, expected to contain 'toolu_01ABC'", result)
	}
	if !contains(result, "read_file") {
		t.Errorf("result = %q, expected to contain 'read_file'", result)
	}
	if !contains(result, "/test.txt") {
		t.Errorf("result = %q, expected to contain '/test.txt'", result)
	}
}

func TestGetCombinedClaudeTools(t *testing.T) {
	mcpTools := []api.ToolDefinition{
		{Name: "mcp_tool_1", Description: "MCP Tool 1"},
		{Name: "mcp_tool_2", Description: "MCP Tool 2"},
	}

	combined := GetCombinedClaudeTools(mcpTools)

	builtInCount := len(GetClaudeToolDefinitions())
	expectedCount := builtInCount + 2

	if len(combined) != expectedCount {
		t.Errorf("GetCombinedClaudeTools() returned %d tools, want %d", len(combined), expectedCount)
	}

	// MCP ツールが含まれていることを確認
	found := 0
	for _, tool := range combined {
		if tool.Name == "mcp_tool_1" || tool.Name == "mcp_tool_2" {
			found++
		}
	}
	if found != 2 {
		t.Errorf("Expected 2 MCP tools in combined list, found %d", found)
	}
}

func TestGetCombinedClaudeTools_NoBP1(t *testing.T) {
	// BP#1 は全モデル共通で省略（BP#2 で tools+system 一体キャッシュ）
	tools := GetCombinedClaudeTools(nil)
	if len(tools) == 0 {
		t.Fatal("expected at least 1 tool")
	}
	last := tools[len(tools)-1]
	if last.CacheControl != nil {
		t.Errorf("expected no cache_control on last tool (BP#1 is always skipped), got %+v", last.CacheControl)
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
