package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// Test helpers

func mockAPIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func assertRequestMethod(t *testing.T, r *http.Request, expected string) {
	t.Helper()
	if r.Method != expected {
		t.Errorf("Expected method %s, got %s", expected, r.Method)
	}
}

func assertJSONContentType(t *testing.T, r *http.Request) {
	t.Helper()
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", contentType)
	}
}

func assertRequestHeader(t *testing.T, r *http.Request, key, expected string) {
	t.Helper()
	value := r.Header.Get(key)
	if value != expected {
		t.Errorf("Expected header %s=%s, got %s", key, expected, value)
	}
}

func errorHandler(status int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(message))
	}
}

func rateLimitHandler(retryAfter string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", retryAfter)
		w.WriteHeader(429)
		_, _ = w.Write([]byte("Rate limited"))
	}
}

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
	if name != "Gemini" {
		t.Errorf("Name() = %v, want 'Gemini'", name)
	}
}

func TestProvider_SupportsImages(t *testing.T) {
	provider := New("test-key")

	supports := provider.SupportsImages()
	if !supports {
		t.Error("SupportsImages() = false, want true (Gemini supports images)")
	}
}

func TestGetGeminiURL(t *testing.T) {
	originalURL := os.Getenv("GEMINI_API_URL")
	defer os.Setenv("GEMINI_API_URL", originalURL)

	t.Run("DefaultURL", func(t *testing.T) {
		os.Unsetenv("GEMINI_API_URL")
		url := getGeminiURL("gemini-2.0-flash-exp")
		expected := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash-exp:streamGenerateContent?alt=sse"
		if url != expected {
			t.Errorf("getGeminiURL() = %q, want %q", url, expected)
		}
	})

	t.Run("CustomURL", func(t *testing.T) {
		customURL := "http://localhost:8080/gemini"
		os.Setenv("GEMINI_API_URL", customURL)
		url := getGeminiURL("any-model")
		if url != customURL {
			t.Errorf("getGeminiURL() = %q, want %q", url, customURL)
		}
	})
}

func TestProvider_ChatWithTools_JSONArray(t *testing.T) {
	// Geminiは SSE 形式でストリーミングレスポンスを返す
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequestMethod(t, r, "POST")
		assertJSONContentType(t, r)
		assertRequestHeader(t, r, "x-goog-api-key", "test-key")

		var req GeminiRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// SSE形式のレスポンス
		w.Header().Set("Content-Type", "text/event-stream")
		resp := GeminiFunctionResponse{
			Candidates: []GeminiFunctionCandidate{{Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "Hello from Gemini"}}}}},
		}
		jsonBytes, _ := json.Marshal(resp)
		fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes))
	})

	originalURL := os.Getenv("GEMINI_API_URL")
	defer os.Setenv("GEMINI_API_URL", originalURL)
	os.Setenv("GEMINI_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	result, err := p.ChatWithTools(context.Background(), "System prompt", history, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Hello from Gemini" {
		t.Errorf("ChatWithTools() = %q, want 'Hello from Gemini'", result)
	}
}

func TestProvider_ChatWithTools_SingleObject(t *testing.T) {
	// 単一オブジェクト形式（配列でない場合）
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		resp := GeminiFunctionResponse{
			Candidates: []GeminiFunctionCandidate{
				{Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "Single response"}}}},
			},
		}
		jsonBytes, _ := json.Marshal(resp)
		fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes))
	})

	originalURL := os.Getenv("GEMINI_API_URL")
	defer os.Setenv("GEMINI_API_URL", originalURL)
	os.Setenv("GEMINI_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "gemini-2.0-flash-exp")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Single response" {
		t.Errorf("ChatWithTools() = %q, want 'Single response'", result)
	}
}

func TestProvider_ChatWithTools_APIError(t *testing.T) {
	server := mockAPIServer(t, errorHandler(500, "Internal Server Error"))

	originalURL := os.Getenv("GEMINI_API_URL")
	defer os.Setenv("GEMINI_API_URL", originalURL)
	os.Setenv("GEMINI_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for 500 status")
	}
}

func TestProvider_ChatWithTools_RateLimit(t *testing.T) {
	server := mockAPIServer(t, rateLimitHandler("60"))

	originalURL := os.Getenv("GEMINI_API_URL")
	defer os.Setenv("GEMINI_API_URL", originalURL)
	os.Setenv("GEMINI_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for rate limit")
	}
}

func TestProvider_ChatWithImage_NoImage(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		resp := GeminiFunctionResponse{
			Candidates: []GeminiFunctionCandidate{{Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "No image response"}}}}},
		}
		jsonBytes, _ := json.Marshal(resp)
		fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes))
	})

	originalURL := os.Getenv("GEMINI_API_URL")
	defer os.Setenv("GEMINI_API_URL", originalURL)
	os.Setenv("GEMINI_API_URL", server.URL)

	p := New("test-key")

	result, err := p.ChatWithImage(context.Background(), "System", nil, "Hello", nil, "")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "No image response" {
		t.Errorf("ChatWithImage() = %q, want 'No image response'", result)
	}
}

func TestProvider_ChatWithImage_WithImage(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		// リクエストボディにinline_dataがあることを確認
		var req GeminiMultimodalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		resp := GeminiFunctionResponse{
			Candidates: []GeminiFunctionCandidate{{Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "Image analysis complete"}}}}},
		}
		jsonBytes, _ := json.Marshal(resp)
		fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes))
	})

	originalURL := os.Getenv("GEMINI_API_URL")
	defer os.Setenv("GEMINI_API_URL", originalURL)
	os.Setenv("GEMINI_API_URL", server.URL)

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

// ===== Function Calling Tests =====

func TestProvider_SetMCPEnabled(t *testing.T) {
	p := New("test-key")

	// 初期状態は false
	if p.mcpEnabled {
		t.Error("mcpEnabled should be false by default")
	}

	// true に設定
	p.SetMCPEnabled(true)
	if !p.mcpEnabled {
		t.Error("mcpEnabled should be true after SetMCPEnabled(true)")
	}

	// false に戻す
	p.SetMCPEnabled(false)
	if p.mcpEnabled {
		t.Error("mcpEnabled should be false after SetMCPEnabled(false)")
	}
}

func TestProvider_ChatWithTools_FunctionCalling(t *testing.T) {
	// Function Calling レスポンスをテスト（SSE形式）
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequestMethod(t, r, "POST")
		assertJSONContentType(t, r)

		// リクエストに tools が含まれていることを確認
		var req GeminiRequestWithTools
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if len(req.Tools) == 0 {
			t.Error("Request should include tools for Function Calling")
		}

		// Function Calling レスポンス（SSE形式）
		w.Header().Set("Content-Type", "text/event-stream")
		resp := GeminiFunctionResponse{
			Candidates: []GeminiFunctionCandidate{
				{
					Content: GeminiFunctionContent{
						Parts: []GeminiFunctionPart{
							{Text: "I'll read that file for you."},
							{FunctionCall: &api.GeminiFunctionCall{
								Name: "read_file",
								Args: map[string]any{"path": "/test/file.txt"},
							}},
						},
					},
				},
			},
		}
		jsonBytes, _ := json.Marshal(resp)
		fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes))
	})

	originalURL := os.Getenv("GEMINI_API_URL")
	originalFC := os.Getenv("GEMINI_FUNCTION_CALLING")
	defer func() {
		os.Setenv("GEMINI_API_URL", originalURL)
		os.Setenv("GEMINI_FUNCTION_CALLING", originalFC)
	}()
	os.Setenv("GEMINI_API_URL", server.URL)
	os.Unsetenv("GEMINI_FUNCTION_CALLING") // デフォルト: 有効

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Read /test/file.txt"}}

	result, err := p.ChatWithTools(context.Background(), "System prompt", history, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// テキストとツール呼び出しが含まれていることを確認
	if result == "" {
		t.Error("ChatWithTools() should return non-empty result")
	}
}

func TestProvider_ChatWithTools_FunctionCallingDisabled(t *testing.T) {
	// GEMINI_FUNCTION_CALLING=0 でテキストモードを使用
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		// テキストモードでは tools が含まれない
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if _, hasTools := req["tools"]; hasTools {
			t.Error("Request should NOT include tools when Function Calling is disabled")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		resp := GeminiFunctionResponse{
			Candidates: []GeminiFunctionCandidate{
				{Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "Text mode response"}}}},
			},
		}
		jsonBytes, _ := json.Marshal(resp)
		fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes))
	})

	originalURL := os.Getenv("GEMINI_API_URL")
	originalFC := os.Getenv("GEMINI_FUNCTION_CALLING")
	defer func() {
		os.Setenv("GEMINI_API_URL", originalURL)
		os.Setenv("GEMINI_FUNCTION_CALLING", originalFC)
	}()
	os.Setenv("GEMINI_API_URL", server.URL)
	os.Setenv("GEMINI_FUNCTION_CALLING", "0") // 無効化

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Text mode response" {
		t.Errorf("ChatWithTools() = %q, want 'Text mode response'", result)
	}
}

func TestProvider_ChatWithTools_WithMCPTools(t *testing.T) {
	// MCPツールが設定されている場合、Function Callingに含まれることを確認
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		// リクエストに tools が含まれていることを確認
		var req GeminiRequestWithTools
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if len(req.Tools) == 0 {
			t.Error("Request should include tools for Function Calling")
		}

		// MCPツールが含まれていることを確認
		foundMCPTool := false
		for _, decl := range req.Tools[0].FunctionDeclarations {
			if decl.Name == "mcp_github_get_issue" {
				foundMCPTool = true
				break
			}
		}
		if !foundMCPTool {
			t.Error("MCP tool 'mcp_github_get_issue' should be included in tools")
		}

		// レスポンス（MCPツール呼び出し）- SSE形式
		w.Header().Set("Content-Type", "text/event-stream")
		resp := GeminiFunctionResponse{
			Candidates: []GeminiFunctionCandidate{
				{
					Content: GeminiFunctionContent{
						Parts: []GeminiFunctionPart{
							{FunctionCall: &api.GeminiFunctionCall{
								Name: "mcp_github_get_issue",
								Args: map[string]any{"owner": "susugadx", "repo": "xelyon-cli", "issue_number": "89"},
							}},
						},
					},
				},
			},
		}
		jsonBytes, _ := json.Marshal(resp)
		fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes))
	})

	originalURL := os.Getenv("GEMINI_API_URL")
	originalFC := os.Getenv("GEMINI_FUNCTION_CALLING")
	defer func() {
		os.Setenv("GEMINI_API_URL", originalURL)
		os.Setenv("GEMINI_FUNCTION_CALLING", originalFC)
	}()
	os.Setenv("GEMINI_API_URL", server.URL)
	os.Unsetenv("GEMINI_FUNCTION_CALLING") // デフォルト: 有効

	p := New("test-key")

	// MCPツールを設定
	mcpTools := []api.GeminiFunctionDeclaration{
		{
			Name:        "mcp_github_get_issue",
			Description: "Get issue details from GitHub",
			Parameters: &api.GeminiParameterSchema{
				Type: "object",
				Properties: map[string]api.GeminiPropertyDef{
					"owner":        {Type: "string", Description: "Repository owner"},
					"repo":         {Type: "string", Description: "Repository name"},
					"issue_number": {Type: "string", Description: "Issue number"},
				},
				Required: []string{"owner", "repo", "issue_number"},
			},
		},
	}
	p.SetMCPTools(mcpTools)

	history := []api.Message{{Role: "user", Content: "Get issue #89 from xelyon-cli"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// MCPツール呼び出しが結果に含まれていることを確認
	if result == "" {
		t.Error("ChatWithTools() should return non-empty result")
	}
}

func TestGetGeminiFunctionCallingURL(t *testing.T) {
	originalURL := os.Getenv("GEMINI_API_URL")
	defer os.Setenv("GEMINI_API_URL", originalURL)

	t.Run("DefaultURL", func(t *testing.T) {
		os.Unsetenv("GEMINI_API_URL")
		url := getGeminiFunctionCallingURL("gemini-2.0-flash-exp")
		// 非ストリーミング（generateContent）を使用
		// NOTE: Gemini APIはPretty-printed JSONを返すため、行単位ストリーミングは不可能
		expected := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash-exp:generateContent"
		if url != expected {
			t.Errorf("getGeminiFunctionCallingURL() = %q, want %q", url, expected)
		}
	})

	t.Run("CustomURL", func(t *testing.T) {
		customURL := "http://localhost:8080/fc"
		os.Setenv("GEMINI_API_URL", customURL)
		url := getGeminiFunctionCallingURL("any-model")
		if url != customURL {
			t.Errorf("getGeminiFunctionCallingURL() = %q, want %q", url, customURL)
		}
	})
}
