package deepseek

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"

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

func TestNew_URLOverride(t *testing.T) {
	// URL環境変数を保存して後でリストア
	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)

	t.Run("DefaultURL", func(t *testing.T) {
		os.Unsetenv("DEEPSEEK_API_URL")
		p := New("test-key")
		if p.APIURL() != defaultDeepSeekURL {
			t.Errorf("apiURL = %q, want %q", p.APIURL(), defaultDeepSeekURL)
		}
	})

	t.Run("CustomURL", func(t *testing.T) {
		customURL := "https://custom.deepseek.api.com/v1"
		os.Setenv("DEEPSEEK_API_URL", customURL)
		p := New("test-key")
		if p.APIURL() != customURL {
			t.Errorf("apiURL = %q, want %q", p.APIURL(), customURL)
		}
	})
}

func TestProvider_Name(t *testing.T) {
	provider := New("test-key")

	name := provider.Name()
	if name != "DeepSeek" {
		t.Errorf("Name() = %v, want 'DeepSeek'", name)
	}
}

func TestProvider_SupportsImages(t *testing.T) {
	provider := New("test-key")

	supports := provider.SupportsImages()
	if supports {
		t.Error("SupportsImages() = true, want false (DeepSeek does not support images)")
	}
}

func TestGetActualModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  string
	}{
		{
			name:  "deepseek-chat (default)",
			model: "deepseek-chat",
			want:  "deepseek-chat",
		},
		{
			name:  "deepseek-coder",
			model: "deepseek-coder",
			want:  "deepseek-coder",
		},
		{
			name:  "deepseek-reasoner",
			model: "deepseek-reasoner",
			want:  "deepseek-reasoner",
		},
		{
			name:  "empty string",
			model: "",
			want:  "deepseek-chat", // デフォルト
		},
		{
			name:  "unknown model",
			model: "unknown-model",
			want:  "deepseek-chat", // デフォルト
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getActualModel(tt.model)
			if got != tt.want {
				t.Errorf("getActualModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
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

func TestProvider_ChatWithTools_RequestValidation(t *testing.T) {
	// モックサーバーを作成（リクエスト検証用）
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

		// リクエストボディを検証
		var req openaicompat.ChatCompletionsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if req.Model != "deepseek-chat" {
			t.Errorf("Model = %q, want 'deepseek-chat'", req.Model)
		}
		if len(req.Messages) != 2 { // system + user
			t.Errorf("Messages count = %d, want 2", len(req.Messages))
		}
		if !req.Stream {
			t.Error("Stream should be true")
		}

		// ストリーミングレスポンス
		streamingHandler([]string{
			`{"choices":[{"delta":{"content":"Test"}}]}`,
			`{"choices":[{"delta":{"content":" response"}}]}`,
		})(w, r)
	})

	// 環境変数でモックサーバーURLを設定
	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	result, err := p.ChatWithTools(context.Background(), "System prompt", history, "deepseek-chat")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Test response" {
		t.Errorf("ChatWithTools() = %q, want 'Test response'", result)
	}
}

func TestProvider_ChatWithTools_Streaming(t *testing.T) {
	chunks := []string{
		`{"choices":[{"delta":{"content":"Hello"}}]}`,
		`{"choices":[{"delta":{"content":" World"}}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "deepseek-chat")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Hello World" {
		t.Errorf("ChatWithTools() = %q, want 'Hello World'", result)
	}
}

func TestProvider_ChatWithTools_ServerError(t *testing.T) {
	server := mockAPIServer(t, errorHandler(500, "Internal Server Error"))

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for 500 status")
	}
}

func TestProvider_ChatWithTools_RateLimitResponse(t *testing.T) {
	server := mockAPIServer(t, rateLimitHandler("1"))

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for rate limit")
	}
}

func TestProvider_ChatWithTools_ContextCancel(t *testing.T) {
	// 長時間待機するハンドラー
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // コンテキストがキャンセルされるまで待機
	})

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 即座にキャンセル

	_, err := p.ChatWithTools(ctx, "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error when context is cancelled")
	}
}

func TestProvider_ChatWithImage_NoImage(t *testing.T) {
	chunks := []string{
		`{"choices":[{"delta":{"content":"No image"}}]}`,
		`{"choices":[{"delta":{"content":" response"}}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := New("test-key")

	// imageがnilの場合
	result, err := p.ChatWithImage(context.Background(), "System", nil, "Hello", nil, "")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "No image response" {
		t.Errorf("ChatWithImage() = %q, want 'No image response'", result)
	}
}

func TestProvider_ChatWithImage_WithImage(t *testing.T) {
	chunks := []string{
		`{"choices":[{"delta":{"content":"Image ignored"}}]}`,
		`{"choices":[{"delta":{"content":" response"}}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := New("test-key")

	// imageがある場合（DeepSeekは画像非対応なので警告を出してテキストのみ送信）
	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png"}
	result, err := p.ChatWithImage(context.Background(), "System", nil, "Describe this", image, "")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "Image ignored response" {
		t.Errorf("ChatWithImage() = %q, want 'Image ignored response'", result)
	}
}

// ============================================
// Function Calling Tests
// ============================================

// TestHandleStreamingResponse_ToolCalls は tool_calls を含むストリーミングレスポンスをテスト
func TestHandleStreamingResponse_ToolCalls(t *testing.T) {
	// tool_calls を含むストリーミングチャンク（分割パターン）
	chunks := []string{
		// tool_call の開始（id と name）
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_abc123","type":"function","function":{"name":"read_file","arguments":""}}]}}]}`,
		// arguments の分割送信
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"pa"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"th\":\"/te"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"st.txt\"}"}}]}}]}`,
		// finish_reason で完了
		`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Read test.txt"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "deepseek-chat")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// 結果に tool call JSON が含まれることを確認
	if result == "" {
		t.Error("ChatWithTools() returned empty string, expected tool call JSON")
	}

	// tool 名が含まれていることを確認
	if !containsString(result, "read_file") {
		t.Errorf("Result should contain tool name 'read_file': %s", result)
	}

	// call_id が含まれていることを確認
	if !containsString(result, "call_abc123") {
		t.Errorf("Result should contain call_id 'call_abc123': %s", result)
	}

	// path 引数が含まれていることを確認
	if !containsString(result, "/test.txt") {
		t.Errorf("Result should contain path '/test.txt': %s", result)
	}
}

// TestHandleStreamingResponse_MultipleToolCalls は複数の tool_calls を含むレスポンスをテスト
func TestHandleStreamingResponse_MultipleToolCalls(t *testing.T) {
	chunks := []string{
		// 2つの tool_call を同時に開始
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_001","type":"function","function":{"name":"read_file","arguments":""}},{"index":1,"id":"call_002","type":"function","function":{"name":"list_dir","arguments":""}}]}}]}`,
		// 各 tool_call の arguments
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"path\":\"file1.txt\"}"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":1,"function":{"arguments":"{\"path\":\".\"}"}}]}}]}`,
		// 完了
		`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "List files and read file1.txt"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "deepseek-chat")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// 両方のツール名が含まれていることを確認
	if !containsString(result, "read_file") {
		t.Errorf("Result should contain 'read_file': %s", result)
	}
	if !containsString(result, "list_dir") {
		t.Errorf("Result should contain 'list_dir': %s", result)
	}

	// 両方の call_id が含まれていることを確認
	if !containsString(result, "call_001") {
		t.Errorf("Result should contain 'call_001': %s", result)
	}
	if !containsString(result, "call_002") {
		t.Errorf("Result should contain 'call_002': %s", result)
	}
}

// TestHandleStreamingResponse_TextOnly は通常のテキスト応答（tool_calls なし）をテスト
func TestHandleStreamingResponse_TextOnly(t *testing.T) {
	chunks := []string{
		`{"choices":[{"delta":{"content":"Hello"}}]}`,
		`{"choices":[{"delta":{"content":" World"}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Say hello"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "deepseek-chat")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	if result != "Hello World" {
		t.Errorf("ChatWithTools() = %q, want 'Hello World'", result)
	}
}

// TestHandleStreamingResponse_TextWithToolCalls はテキストと tool_calls が混在するレスポンスをテスト
func TestHandleStreamingResponse_TextWithToolCalls(t *testing.T) {
	chunks := []string{
		// まずテキストを出力
		`{"choices":[{"delta":{"content":"Let me read that file."}}]}`,
		// 次に tool_call
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_mixed","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"test.go\"}"}}]}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	defer os.Setenv("DEEPSEEK_API_URL", originalURL)
	os.Setenv("DEEPSEEK_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Read test.go"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "deepseek-chat")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// テキストと tool_call の両方が含まれていることを確認
	if !containsString(result, "Let me read that file.") {
		t.Errorf("Result should contain text content: %s", result)
	}
	if !containsString(result, "read_file") {
		t.Errorf("Result should contain tool call: %s", result)
	}
}

// TestSetMCPTools は MCP ツールの設定をテスト
func TestSetMCPTools(t *testing.T) {
	p := New("test-key")

	// 初期状態では mcpTools は nil
	if p.mcpTools != nil {
		t.Error("Initial mcpTools should be nil")
	}

	// MCP ツールを設定
	mcpTools := []api.ToolDefinition{
		{
			Name:        "mcp_tool_1",
			Description: "First MCP tool",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "mcp_tool_2",
			Description: "Second MCP tool",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"param1": map[string]interface{}{"type": "string"},
				},
			},
		},
	}

	p.SetMCPTools(mcpTools)

	// 設定されたことを確認
	if len(p.mcpTools) != 2 {
		t.Errorf("mcpTools length = %d, want 2", len(p.mcpTools))
	}

	if p.mcpTools[0].Name != "mcp_tool_1" {
		t.Errorf("mcpTools[0].Name = %q, want 'mcp_tool_1'", p.mcpTools[0].Name)
	}

	if p.mcpTools[1].Name != "mcp_tool_2" {
		t.Errorf("mcpTools[1].Name = %q, want 'mcp_tool_2'", p.mcpTools[1].Name)
	}
}

// TestSetMCPTools_Empty は空の MCP ツールリストの設定をテスト
func TestSetMCPTools_Empty(t *testing.T) {
	p := New("test-key")

	p.SetMCPTools([]api.ToolDefinition{})

	if p.mcpTools == nil {
		t.Error("mcpTools should not be nil after setting empty slice")
	}
	if len(p.mcpTools) != 0 {
		t.Errorf("mcpTools length = %d, want 0", len(p.mcpTools))
	}
}

// TestGetCombinedTools は組み込みツールと MCP ツールの結合をテスト
func TestGetCombinedTools(t *testing.T) {
	// openai.GetCombinedOpenAITools を使用して確認
	// （deepseek は openai パッケージの関数を流用）

	// MCP ツールなし
	toolsNoMCP := openai.GetCombinedOpenAITools(nil)
	builtinCount := len(toolsNoMCP)

	if builtinCount == 0 {
		t.Error("Built-in tools count should be > 0")
	}

	// MCP ツールあり
	mcpTools := []api.ToolDefinition{
		{
			Name:        "custom_mcp_tool",
			Description: "A custom tool",
			Parameters:  map[string]interface{}{"type": "object"},
		},
	}

	toolsWithMCP := openai.GetCombinedOpenAITools(mcpTools)

	expectedCount := builtinCount + 1
	if len(toolsWithMCP) != expectedCount {
		t.Errorf("Combined tools count = %d, want %d", len(toolsWithMCP), expectedCount)
	}

	// MCP ツールが含まれていることを確認
	found := false
	for _, tool := range toolsWithMCP {
		if tool.Function != nil && tool.Function.Name == "custom_mcp_tool" {
			found = true
			break
		}
	}
	if !found {
		t.Error("MCP tool 'custom_mcp_tool' not found in combined tools")
	}
}

// TestGetCombinedTools_BuiltinToolsPresent は組み込みツールが含まれていることを確認
func TestGetCombinedTools_BuiltinToolsPresent(t *testing.T) {
	tools := openai.GetCombinedOpenAITools(nil)

	// 必須の組み込みツールが含まれていることを確認
	requiredTools := []string{
		"read_file",
		"write_file",
		"str_replace",
		"bash",
		"list_dir",
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		if tool.Function != nil {
			toolNames[tool.Function.Name] = true
		}
	}

	for _, required := range requiredTools {
		if !toolNames[required] {
			t.Errorf("Required built-in tool %q not found", required)
		}
	}
}

// TestFunctionCalling_RequestFormat は Function Calling 有効時のリクエスト形式をテスト
func TestFunctionCalling_RequestFormat(t *testing.T) {
	var capturedRequest openaicompat.ChatCompletionsRequest

	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedRequest); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// ストリーミングレスポンス
		streamingHandler([]string{
			`{"choices":[{"delta":{"content":"OK"}}]}`,
		})(w, r)
	})

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	originalFC := os.Getenv("DEEPSEEK_FUNCTION_CALLING")
	defer func() {
		os.Setenv("DEEPSEEK_API_URL", originalURL)
		os.Setenv("DEEPSEEK_FUNCTION_CALLING", originalFC)
	}()

	os.Setenv("DEEPSEEK_API_URL", server.URL)
	os.Unsetenv("DEEPSEEK_FUNCTION_CALLING") // デフォルト（有効）

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "deepseek-chat")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// Tools が設定されていることを確認
	if len(capturedRequest.Tools) == 0 {
		t.Error("Request should contain tools when Function Calling is enabled")
	}

	// ToolChoice が "auto" であることを確認
	if capturedRequest.ToolChoice != "auto" {
		t.Errorf("ToolChoice = %q, want 'auto'", capturedRequest.ToolChoice)
	}
}

// TestFunctionCalling_Disabled は DEEPSEEK_FUNCTION_CALLING=0 でツールが含まれないことを確認
func TestFunctionCalling_Disabled(t *testing.T) {
	var capturedRequest openaicompat.ChatCompletionsRequest

	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedRequest); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		streamingHandler([]string{
			`{"choices":[{"delta":{"content":"OK"}}]}`,
		})(w, r)
	})

	originalURL := os.Getenv("DEEPSEEK_API_URL")
	originalFC := os.Getenv("DEEPSEEK_FUNCTION_CALLING")
	defer func() {
		os.Setenv("DEEPSEEK_API_URL", originalURL)
		if originalFC != "" {
			os.Setenv("DEEPSEEK_FUNCTION_CALLING", originalFC)
		} else {
			os.Unsetenv("DEEPSEEK_FUNCTION_CALLING")
		}
	}()

	os.Setenv("DEEPSEEK_API_URL", server.URL)
	os.Setenv("DEEPSEEK_FUNCTION_CALLING", "0") // 無効化

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "deepseek-chat")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// Tools が設定されていないことを確認
	if len(capturedRequest.Tools) != 0 {
		t.Errorf("Request should not contain tools when Function Calling is disabled, got %d tools", len(capturedRequest.Tools))
	}

	// ToolChoice も空であることを確認
	if capturedRequest.ToolChoice != "" {
		t.Errorf("ToolChoice should be empty when disabled, got %q", capturedRequest.ToolChoice)
	}
}

func TestProvider_ToolChoiceAndReasoningAccessors(t *testing.T) {
	p := New("test-key")

	if got := p.LastReasoningContent(); got != "" {
		t.Fatalf("LastReasoningContent() = %q, want empty", got)
	}

	p.lastReasoningContent = "thinking..."
	if got := p.LastReasoningContent(); got != "thinking..." {
		t.Fatalf("LastReasoningContent() = %q, want %q", got, "thinking...")
	}

	p.SetToolChoice("read_file")
	if p.toolChoice == nil || *p.toolChoice != "read_file" {
		t.Fatalf("toolChoice = %v, want read_file", p.toolChoice)
	}
	p.ClearToolChoice()
	if p.toolChoice != nil {
		t.Fatal("toolChoice should be nil after ClearToolChoice")
	}

	var usage api.Usage
	p.SetUsageCallback(func(u api.Usage) {
		usage = u
	})
	if p.usageCallback == nil {
		t.Fatal("usageCallback should be set")
	}
	p.usageCallback(api.Usage{InputTokens: 12})
	if usage.InputTokens != 12 {
		t.Fatalf("usage = %+v, want input=12", usage)
	}
}

// containsString は文字列に部分文字列が含まれるかチェック
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
