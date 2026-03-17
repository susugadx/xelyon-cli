package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"

	// ツール登録のための blank import
	_ "github.com/susugadx/xelyon-cli/internal/tools/dev"
	_ "github.com/susugadx/xelyon-cli/internal/tools/file"
	_ "github.com/susugadx/xelyon-cli/internal/tools/lsp"
	_ "github.com/susugadx/xelyon-cli/internal/tools/search"
)

// mockAPIServer creates a test HTTP server
func mockAPIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// streamingHandler returns a handler that sends SSE events
func streamingHandler(chunks []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}

		fmt.Fprintf(w, "data: [DONE]\n\n")
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

func TestNewOpenAIProvider(t *testing.T) {
	apiKey := "test-api-key"
	provider := New(apiKey)

	if provider == nil {
		t.Fatal("New() returned nil")
	}
}

func TestOpenAIProvider_Name(t *testing.T) {
	provider := New("test-key")

	name := provider.Name()
	if name != "OpenAI" {
		t.Errorf("Name() = %v, want 'OpenAI'", name)
	}
}

func TestOpenAIProvider_SupportsImages(t *testing.T) {
	provider := New("test-key")

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
		p := New("test-key")
		if p.APIURL != defaultOpenAIURL {
			t.Errorf("apiURL = %q, want %q", p.APIURL, defaultOpenAIURL)
		}
	})

	t.Run("CustomURL", func(t *testing.T) {
		customURL := "https://custom.openai.api.com/v1"
		os.Setenv("OPENAI_API_URL", customURL)
		p := New("test-key")
		if p.APIURL != customURL {
			t.Errorf("apiURL = %q, want %q", p.APIURL, customURL)
		}
	})
}

func TestOpenAIProvider_ChatWithTools_NonStreaming(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization = %q, want 'Bearer test-key'", r.Header.Get("Authorization"))
		}

		var req api.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if req.Model != "gpt-4-turbo" {
			t.Errorf("Model = %q, want 'gpt-4-turbo'", req.Model)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := api.ChatResponse{
			Choices: []api.Choice{
				{Message: api.Message{Content: "Test response from OpenAI"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("OPENAI_API_URL")
	defer os.Setenv("OPENAI_API_URL", originalURL)
	os.Setenv("OPENAI_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	result, err := p.ChatWithTools(context.Background(), "System prompt", history, "gpt-4-turbo")
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

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "gpt-4-turbo")
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

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for 500 status")
	}
}

func TestOpenAIProvider_ChatWithTools_RateLimit(t *testing.T) {
	server := mockAPIServer(t, rateLimitHandler("1"))

	originalURL := os.Getenv("OPENAI_API_URL")
	defer os.Setenv("OPENAI_API_URL", originalURL)
	os.Setenv("OPENAI_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for rate limit")
	}
}

func TestOpenAIProvider_ChatWithImage_NoImage(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := api.ChatResponse{
			Choices: []api.Choice{
				{Message: api.Message{Content: "No image response"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("OPENAI_API_URL")
	defer os.Setenv("OPENAI_API_URL", originalURL)
	os.Setenv("OPENAI_API_URL", server.URL)

	p := New("test-key")

	result, err := p.ChatWithImage(context.Background(), "System", nil, "Hello", nil, "gpt-4-turbo")
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
		var req MultimodalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if len(req.Messages) < 2 { // system + multimodal message
			t.Errorf("Messages count = %d, want at least 2", len(req.Messages))
		}

		w.Header().Set("Content-Type", "application/json")
		resp := api.ChatResponse{
			Choices: []api.Choice{
				{Message: api.Message{Content: "Image analysis complete"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("OPENAI_API_URL")
	defer os.Setenv("OPENAI_API_URL", originalURL)
	os.Setenv("OPENAI_API_URL", server.URL)

	p := New("test-key")
	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png"}

	result, err := p.ChatWithImage(context.Background(), "System", nil, "Describe this", image, "gpt-4-turbo")
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

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	result, err := p.chatWithResponses(context.Background(), "System", history, "gpt-5.2-codex")
	if err != nil {
		t.Fatalf("chatWithResponses() error = %v", err)
	}
	if result != "Hello from Codex" {
		t.Errorf("chatWithResponses() = %q, want 'Hello from Codex'", result)
	}
}

func TestOpenAIProvider_ResponsesAPI_ErrorEvent(t *testing.T) {
	chunks := []string{
		`{"type":"response.created","response":{"id":"resp_err"}}`,
		`{"type":"error","error":{"code":"server_error","message":"temporary upstream failure"}}`,
	}
	server := mockAPIServer(t, streamingHandler(chunks))

	originalURL := os.Getenv("OPENAI_RESPONSES_URL")
	defer os.Setenv("OPENAI_RESPONSES_URL", originalURL)
	os.Setenv("OPENAI_RESPONSES_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.chatWithResponses(context.Background(), "System", history, "gpt-5.2-codex")
	if err == nil {
		t.Fatal("chatWithResponses() should return error for error event")
	}
	if got := err.Error(); !strings.Contains(got, "temporary upstream failure") {
		t.Fatalf("chatWithResponses() error = %q, want to contain temporary upstream failure", got)
	}
}

func TestOpenAIProvider_ResponsesAPI_RequestFormat(t *testing.T) {
	var receivedReq ResponsesRequest

	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Method = %q, want POST", r.Method)
		}

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

	p := New("test-key")
	history := []api.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
	}

	_, _ = p.chatWithResponses(context.Background(), "You are helpful", history, "gpt-5.2-codex")

	// リクエスト形式を確認
	if receivedReq.Model != "gpt-5.2-codex" {
		t.Errorf("Model = %q, want 'gpt-5.2-codex'", receivedReq.Model)
	}
	// Instructions は空（developer メッセージとして Input に含まれる）
	if receivedReq.Instructions != "" {
		t.Errorf("Instructions = %q, want empty (should be in Input as developer message)", receivedReq.Instructions)
	}
	if !receivedReq.Stream {
		t.Error("Stream should be true")
	}
	// Input の先頭が developer メッセージであることを確認
	inputItems, ok := receivedReq.Input.([]interface{})
	if !ok || len(inputItems) == 0 {
		t.Fatal("Input should be non-empty array")
	}
	firstItem, ok := inputItems[0].(map[string]interface{})
	if !ok {
		t.Fatal("First input item should be a map")
	}
	if firstItem["role"] != "developer" {
		t.Errorf("First input item role = %q, want 'developer'", firstItem["role"])
	}
	if firstItem["content"] != "You are helpful" {
		t.Errorf("First input item content = %q, want 'You are helpful'", firstItem["content"])
	}
}

func TestOpenAIProvider_ResponsesAPI_ToolResultUsesPreviousResponseID(t *testing.T) {
	var requests []ResponsesRequest

	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req ResponsesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		requests = append(requests, req)

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("Streaming unsupported")
		}
		if len(requests) == 1 {
			fmt.Fprintf(w, "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_first\"}}\n\n")
			fmt.Fprintf(w, "data: {\"type\":\"response.output_item.added\",\"item\":{\"type\":\"function_call\",\"call_id\":\"call_1\",\"name\":\"read_file\"}}\n\n")
			fmt.Fprintf(w, "data: {\"type\":\"response.function_call_arguments.done\",\"item_id\":\"dummy\",\"output_index\":0,\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\":\\\"a.txt\\\"}\",\"call_id\":\"call_1\"}\n\n")
		} else {
			fmt.Fprintf(w, "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_second\"}}\n\n")
			fmt.Fprintf(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"done\"}\n\n")
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	})

	originalURL := os.Getenv("OPENAI_RESPONSES_URL")
	defer os.Setenv("OPENAI_RESPONSES_URL", originalURL)
	os.Setenv("OPENAI_RESPONSES_URL", server.URL)

	p := New("test-key")

	firstHistory := []api.Message{{Role: "user", Content: "read a.txt"}}
	_, _ = p.chatWithResponses(context.Background(), "You are helpful", firstHistory, "gpt-5.2-codex")

	secondHistory := []api.Message{
		{Role: "assistant", ToolCalls: []api.OpenAIToolCall{{ID: "call_1", Type: "function", Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{"path":"a.txt"}`}}}},
		{Role: "tool", ToolCallID: "call_1", Content: "file-a"},
		{Role: "tool", ToolCallID: "call_2", Content: "file-b"},
	}
	_, _ = p.chatWithResponses(context.Background(), "You are helpful", secondHistory, "gpt-5.2-codex")

	if len(requests) != 2 {
		t.Fatalf("request count = %d, want 2", len(requests))
	}

	secondReq := requests[1]
	if secondReq.PreviousResponseID != "resp_first" {
		t.Errorf("PreviousResponseID = %q, want 'resp_first'", secondReq.PreviousResponseID)
	}
	inputItems, ok := secondReq.Input.([]interface{})
	if !ok {
		t.Fatalf("Input type = %T, want []interface{}", secondReq.Input)
	}
	if len(inputItems) != 2 {
		t.Fatalf("Input length = %d, want 2", len(inputItems))
	}

	first, ok := inputItems[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Input[0] type = %T, want map[string]interface{}", inputItems[0])
	}
	if first["type"] != "function_call_output" {
		t.Errorf("Input[0].type = %v, want function_call_output", first["type"])
	}
	if first["call_id"] != "call_1" {
		t.Errorf("Input[0].call_id = %v, want call_1", first["call_id"])
	}
	if first["output"] != "file-a" {
		t.Errorf("Input[0].output = %v, want file-a", first["output"])
	}

	second, ok := inputItems[1].(map[string]interface{})
	if !ok {
		t.Fatalf("Input[1] type = %T, want map[string]interface{}", inputItems[1])
	}
	if second["type"] != "function_call_output" {
		t.Errorf("Input[1].type = %v, want function_call_output", second["type"])
	}
	if second["call_id"] != "call_2" {
		t.Errorf("Input[1].call_id = %v, want call_2", second["call_id"])
	}
	if second["output"] != "file-b" {
		t.Errorf("Input[1].output = %v, want file-b", second["output"])
	}

	if _, hasRole := first["role"]; hasRole {
		t.Error("Input[0] should not include role for function_call_output")
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

func TestLevelToReasoningEffort(t *testing.T) {
	tests := []struct {
		level string
		want  string
	}{
		{"low", "low"},
		{"medium", "medium"},
		{"high", "high"},
		{"xhigh", "xhigh"},
		{"unknown", "medium"}, // default
		{"", "medium"},        // default
		{"invalid", "medium"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			got := LevelToReasoningEffort(tt.level)
			if got != tt.want {
				t.Errorf("LevelToReasoningEffort(%q) = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}

func TestOpenAIProvider_SetGetResponseID(t *testing.T) {
	p := New("test-key")

	// 初期状態: 空
	if p.GetResponseID() != "" {
		t.Errorf("GetResponseID() = %q, want empty", p.GetResponseID())
	}
	if p.HasCachedResponseID() {
		t.Error("HasCachedResponseID() = true, want false")
	}

	// Set
	p.SetResponseID("resp_abc123")
	if p.GetResponseID() != "resp_abc123" {
		t.Errorf("GetResponseID() = %q, want 'resp_abc123'", p.GetResponseID())
	}
	if !p.HasCachedResponseID() {
		t.Error("HasCachedResponseID() = false, want true")
	}

	// Clear
	p.ClearResponseID()
	if p.GetResponseID() != "" {
		t.Errorf("GetResponseID() after clear = %q, want empty", p.GetResponseID())
	}
	if p.HasCachedResponseID() {
		t.Error("HasCachedResponseID() after clear = true, want false")
	}

	// Set again
	p.SetResponseID("resp_xyz")
	if p.GetResponseID() != "resp_xyz" {
		t.Errorf("GetResponseID() = %q, want 'resp_xyz'", p.GetResponseID())
	}

	// ClearCache also clears
	p.ClearCache()
	if p.GetResponseID() != "" {
		t.Errorf("GetResponseID() after ClearCache = %q, want empty", p.GetResponseID())
	}
}

func TestIsCodexModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"gpt-5.2-codex", true},
		{"gpt-5.1-codex", true},
		{"gpt-5.1-codex-max", true},
		{"gpt-5-codex", true},
		{"GPT-5.2-CODEX", true}, // case insensitive
		{"gpt-5.2-Codex", true}, // mixed case
		{"gpt-4o", false},
		{"gpt-5.2", false},
		{"claude-3-opus", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := isCodexModel(tt.model)
			if got != tt.want {
				t.Errorf("isCodexModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}
