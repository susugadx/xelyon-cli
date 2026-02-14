package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"

	// ツール登録のための blank import
	_ "github.com/susugadx/xelyon-cli/internal/tools/dev"
	_ "github.com/susugadx/xelyon-cli/internal/tools/file"
	_ "github.com/susugadx/xelyon-cli/internal/tools/lsp"
	_ "github.com/susugadx/xelyon-cli/internal/tools/search"
)

func TestNew(t *testing.T) {
	baseURL := "http://localhost:11434"
	provider := New(baseURL)

	if provider == nil {
		t.Fatal("New() returned nil")
	}
}

func TestProvider_Name(t *testing.T) {
	provider := New("http://localhost:11434")

	name := provider.Name()
	if name != "Ollama" {
		t.Errorf("Name() = %v, want 'Ollama'", name)
	}
}

func TestProvider_SupportsImages(t *testing.T) {
	provider := New("http://localhost:11434")

	supports := provider.SupportsImages()
	if supports {
		t.Error("SupportsImages() = true, want false (Ollama does not support images)")
	}
}

func TestNew_DefaultURL(t *testing.T) {
	p := New("")
	if p.BaseURL() != "http://localhost:11434" {
		t.Errorf("baseURL = %q, want 'http://localhost:11434'", p.BaseURL())
	}
}

func TestNew_CustomURL(t *testing.T) {
	customURL := "http://custom-ollama:8080"
	p := New(customURL)
	if p.BaseURL() != customURL {
		t.Errorf("baseURL = %q, want %q", p.BaseURL(), customURL)
	}
}

// Helper functions for testing

func mockAPIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
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

// ollamaStreamingHandler はOllama形式のJSON Linesストリーミングハンドラー
func ollamaStreamingHandler(texts []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		for _, text := range texts {
			resp := OllamaStreamResponse{
				Message: OllamaMessageContent{Content: text},
				Done:    false,
			}
			data, _ := json.Marshal(resp)
			fmt.Fprintln(w, string(data))
			flusher.Flush()
		}

		// 終了レスポンス
		doneResp := OllamaStreamResponse{Done: true}
		data, _ := json.Marshal(doneResp)
		fmt.Fprintln(w, string(data))
		flusher.Flush()
	}
}

func TestProvider_ChatWithTools_Streaming(t *testing.T) {
	server := mockAPIServer(t, ollamaStreamingHandler([]string{"Hello", " from", " Ollama"}))

	p := New(server.URL)
	history := []api.Message{{Role: "user", Content: "Hi"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "llama3")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Hello from Ollama" {
		t.Errorf("ChatWithTools() = %q, want 'Hello from Ollama'", result)
	}
}

func TestProvider_ChatWithTools_APIError(t *testing.T) {
	server := mockAPIServer(t, errorHandler(500, "Internal Server Error"))

	p := New(server.URL)
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for 500 status")
	}
}

func TestProvider_ChatWithTools_RateLimit(t *testing.T) {
	server := mockAPIServer(t, rateLimitHandler("1"))

	p := New(server.URL)
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for rate limit")
	}
}

func TestProvider_ChatWithImage(t *testing.T) {
	server := mockAPIServer(t, ollamaStreamingHandler([]string{"Image", " ignored"}))

	p := New(server.URL)
	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png"}

	// Ollamaは画像非対応なのでテキストのみ送信
	result, err := p.ChatWithImage(context.Background(), "System", nil, "Describe this", image, "")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "Image ignored" {
		t.Errorf("ChatWithImage() = %q, want 'Image ignored'", result)
	}
}

func TestProvider_ListModels_Success(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("URL path = %q, want '/api/tags'", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := OllamaTagsResponse{
			Models: []OllamaModel{
				{Name: "llama3:latest"},
				{Name: "mistral:latest"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	p := New(server.URL)
	models, err := p.ListModels()
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if len(models) != 2 {
		t.Errorf("ListModels() returned %d models, want 2", len(models))
	}
	if models[0] != "llama3:latest" {
		t.Errorf("models[0] = %q, want 'llama3:latest'", models[0])
	}
}

func TestProvider_ListModels_APIError(t *testing.T) {
	server := mockAPIServer(t, errorHandler(500, "Internal Server Error"))

	p := New(server.URL)
	_, err := p.ListModels()
	if err == nil {
		t.Error("ListModels() should return error for 500 status")
	}
}

// ===== Function Calling Tests =====

func TestSetMCPTools(t *testing.T) {
	p := New("http://localhost:11434")

	tools := []api.ToolDefinition{
		{
			Name:        "test_tool",
			Description: "A test tool",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	p.SetMCPTools(tools)

	if len(p.mcpTools) != 1 {
		t.Errorf("mcpTools length = %d, want 1", len(p.mcpTools))
	}
	if p.mcpTools[0].Name != "test_tool" {
		t.Errorf("mcpTools[0].Name = %q, want 'test_tool'", p.mcpTools[0].Name)
	}
}

func TestFunctionCalling_Disabled(t *testing.T) {
	// 環境変数で無効化
	t.Setenv("OLLAMA_FUNCTION_CALLING", "0")

	var receivedRequest OllamaRequest
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedRequest)

		w.Header().Set("Content-Type", "application/x-ndjson")
		resp := OllamaStreamResponse{Done: true}
		data, _ := json.Marshal(resp)
		fmt.Fprintln(w, string(data))
	})

	p := New(server.URL)
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, _ = p.ChatWithTools(context.Background(), "System", history, "llama3")

	// Tools が設定されていないことを確認
	if len(receivedRequest.Tools) != 0 {
		t.Errorf("Tools should be empty when disabled, got %d tools", len(receivedRequest.Tools))
	}
}

// ollamaToolCallsHandler はtool_callsを含むOllama形式のストリーミングハンドラー
func ollamaToolCallsHandler(toolCalls []api.OpenAIToolCall) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// tool_calls を含むレスポンス
		resp := OllamaStreamResponse{
			Message: OllamaMessageContent{ToolCalls: toolCalls},
			Done:    false,
		}
		data, _ := json.Marshal(resp)
		fmt.Fprintln(w, string(data))
		flusher.Flush()

		// 終了レスポンス
		doneResp := OllamaStreamResponse{Done: true}
		data, _ = json.Marshal(doneResp)
		fmt.Fprintln(w, string(data))
		flusher.Flush()
	}
}

func TestProvider_ChatWithTools_ToolCalls(t *testing.T) {
	toolCalls := []api.OpenAIToolCall{
		{
			ID:   "call_123",
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      "read_file",
				Arguments: `{"path":"README.md"}`,
			},
		},
	}

	server := mockAPIServer(t, ollamaToolCallsHandler(toolCalls))

	p := New(server.URL)
	history := []api.Message{{Role: "user", Content: "Read README.md"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "llama3")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// tool_callsがJSON形式で返される
	if result == "" {
		t.Error("ChatWithTools() returned empty result")
	}

	// read_file ツール呼び出しが含まれる
	if !contains(result, "read_file") {
		t.Errorf("result should contain 'read_file', got %q", result)
	}
	if !contains(result, "call_123") {
		t.Errorf("result should contain 'call_123', got %q", result)
	}
}

func TestProvider_ChatWithTools_MultipleToolCalls(t *testing.T) {
	toolCalls := []api.OpenAIToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      "read_file",
				Arguments: `{"path":"file1.txt"}`,
			},
		},
		{
			ID:   "call_2",
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      "read_file",
				Arguments: `{"path":"file2.txt"}`,
			},
		},
	}

	server := mockAPIServer(t, ollamaToolCallsHandler(toolCalls))

	p := New(server.URL)
	history := []api.Message{{Role: "user", Content: "Read files"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "llama3")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// 両方のtool_callsが含まれる
	if !contains(result, "call_1") {
		t.Errorf("result should contain 'call_1', got %q", result)
	}
	if !contains(result, "call_2") {
		t.Errorf("result should contain 'call_2', got %q", result)
	}
}

// ollamaTextAndToolCallsHandler はテキストとtool_callsを含むハンドラー
func ollamaTextAndToolCallsHandler(texts []string, toolCalls []api.OpenAIToolCall) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// テキストコンテンツ
		for _, text := range texts {
			resp := OllamaStreamResponse{
				Message: OllamaMessageContent{Content: text},
				Done:    false,
			}
			data, _ := json.Marshal(resp)
			fmt.Fprintln(w, string(data))
			flusher.Flush()
		}

		// tool_calls を含むレスポンス
		resp := OllamaStreamResponse{
			Message: OllamaMessageContent{ToolCalls: toolCalls},
			Done:    false,
		}
		data, _ := json.Marshal(resp)
		fmt.Fprintln(w, string(data))
		flusher.Flush()

		// 終了レスポンス
		doneResp := OllamaStreamResponse{Done: true}
		data, _ = json.Marshal(doneResp)
		fmt.Fprintln(w, string(data))
		flusher.Flush()
	}
}

func TestProvider_ChatWithTools_TextAndToolCalls(t *testing.T) {
	toolCalls := []api.OpenAIToolCall{
		{
			ID:   "call_abc",
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      "search_code",
				Arguments: `{"pattern":"TODO"}`,
			},
		},
	}

	server := mockAPIServer(t, ollamaTextAndToolCallsHandler([]string{"I'll search", " for TODOs"}, toolCalls))

	p := New(server.URL)
	history := []api.Message{{Role: "user", Content: "Find TODOs"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "llama3")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// テキストとtool_calls両方が含まれる
	if !contains(result, "I'll search for TODOs") {
		t.Errorf("result should contain text content, got %q", result)
	}
	if !contains(result, "search_code") {
		t.Errorf("result should contain 'search_code', got %q", result)
	}
}

// contains はsがsubstrを含むか確認するヘルパー
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
