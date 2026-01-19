package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

func TestNewGeminiProvider(t *testing.T) {
	apiKey := "test-api-key"
	provider := NewGeminiProvider(apiKey)

	if provider == nil {
		t.Fatal("NewGeminiProvider() returned nil")
	}
}

func TestGeminiProvider_Name(t *testing.T) {
	provider := NewGeminiProvider("test-key")

	name := provider.Name()
	if name != "Gemini" {
		t.Errorf("Name() = %v, want 'Gemini'", name)
	}
}

func TestGeminiProvider_SupportsImages(t *testing.T) {
	provider := NewGeminiProvider("test-key")

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
		expected := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash-exp:streamGenerateContent"
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

func TestGeminiProvider_ChatWithTools_JSONArray(t *testing.T) {
	// Geminiは JSON 配列形式でストリーミングレスポンスを返す
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequestMethod(t, r, "POST")
		assertJSONContentType(t, r)
		assertRequestHeader(t, r, "x-goog-api-key", "test-key")

		var req GeminiRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// JSON配列形式のレスポンス
		w.Header().Set("Content-Type", "application/json")
		responses := []GeminiResponse{
			{Candidates: []GeminiCandidate{{Content: GeminiContent{Parts: []GeminiPart{{Text: "Hello from Gemini"}}}}}},
		}
		_ = json.NewEncoder(w).Encode(responses)
	})

	originalURL := os.Getenv("GEMINI_API_URL")
	defer os.Setenv("GEMINI_API_URL", originalURL)
	os.Setenv("GEMINI_API_URL", server.URL)

	p := NewGeminiProvider("test-key")
	history := []Message{{Role: "user", Content: "Hello"}}

	result, err := p.ChatWithTools(context.Background(), "System prompt", history, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Hello from Gemini" {
		t.Errorf("ChatWithTools() = %q, want 'Hello from Gemini'", result)
	}
}

func TestGeminiProvider_ChatWithTools_SingleObject(t *testing.T) {
	// 単一オブジェクト形式（配列でない場合）
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := GeminiResponse{
			Candidates: []GeminiCandidate{
				{Content: GeminiContent{Parts: []GeminiPart{{Text: "Single response"}}}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("GEMINI_API_URL")
	defer os.Setenv("GEMINI_API_URL", originalURL)
	os.Setenv("GEMINI_API_URL", server.URL)

	p := NewGeminiProvider("test-key")
	history := []Message{{Role: "user", Content: "Hi"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "gemini-2.0-flash-exp")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Single response" {
		t.Errorf("ChatWithTools() = %q, want 'Single response'", result)
	}
}

func TestGeminiProvider_ChatWithTools_APIError(t *testing.T) {
	server := mockAPIServer(t, errorHandler(500, "Internal Server Error"))

	originalURL := os.Getenv("GEMINI_API_URL")
	defer os.Setenv("GEMINI_API_URL", originalURL)
	os.Setenv("GEMINI_API_URL", server.URL)

	p := NewGeminiProvider("test-key")
	history := []Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for 500 status")
	}
}

func TestGeminiProvider_ChatWithTools_RateLimit(t *testing.T) {
	server := mockAPIServer(t, rateLimitHandler("60"))

	originalURL := os.Getenv("GEMINI_API_URL")
	defer os.Setenv("GEMINI_API_URL", originalURL)
	os.Setenv("GEMINI_API_URL", server.URL)

	p := NewGeminiProvider("test-key")
	history := []Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for rate limit")
	}
}

func TestGeminiProvider_ChatWithImage_NoImage(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		responses := []GeminiResponse{
			{Candidates: []GeminiCandidate{{Content: GeminiContent{Parts: []GeminiPart{{Text: "No image response"}}}}}},
		}
		_ = json.NewEncoder(w).Encode(responses)
	})

	originalURL := os.Getenv("GEMINI_API_URL")
	defer os.Setenv("GEMINI_API_URL", originalURL)
	os.Setenv("GEMINI_API_URL", server.URL)

	p := NewGeminiProvider("test-key")

	result, err := p.ChatWithImage(context.Background(), "System", nil, "Hello", nil, "")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "No image response" {
		t.Errorf("ChatWithImage() = %q, want 'No image response'", result)
	}
}

func TestGeminiProvider_ChatWithImage_WithImage(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		// リクエストボディにinline_dataがあることを確認
		var req GeminiMultimodalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		responses := []GeminiResponse{
			{Candidates: []GeminiCandidate{{Content: GeminiContent{Parts: []GeminiPart{{Text: "Image analysis complete"}}}}}},
		}
		_ = json.NewEncoder(w).Encode(responses)
	})

	originalURL := os.Getenv("GEMINI_API_URL")
	defer os.Setenv("GEMINI_API_URL", originalURL)
	os.Setenv("GEMINI_API_URL", server.URL)

	p := NewGeminiProvider("test-key")
	image := &ImageData{Base64: "dGVzdA==", MediaType: "image/png"}

	result, err := p.ChatWithImage(context.Background(), "System", nil, "Describe this", image, "")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "Image analysis complete" {
		t.Errorf("ChatWithImage() = %q, want 'Image analysis complete'", result)
	}
}

// ===== Function Calling Tests =====

func TestGeminiProvider_SetMCPEnabled(t *testing.T) {
	p := NewGeminiProvider("test-key")

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

func TestGeminiProvider_ChatWithTools_FunctionCalling(t *testing.T) {
	// Function Calling レスポンスをテスト
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

		// Function Calling レスポンス
		w.Header().Set("Content-Type", "application/json")
		resp := GeminiFunctionResponse{
			Candidates: []GeminiFunctionCandidate{
				{
					Content: GeminiFunctionContent{
						Parts: []GeminiFunctionPart{
							{Text: "I'll read that file for you."},
							{FunctionCall: &GeminiFunctionCall{
								Name: "read_file",
								Args: map[string]any{"path": "/test/file.txt"},
							}},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("GEMINI_API_URL")
	originalFC := os.Getenv("GEMINI_FUNCTION_CALLING")
	defer func() {
		os.Setenv("GEMINI_API_URL", originalURL)
		os.Setenv("GEMINI_FUNCTION_CALLING", originalFC)
	}()
	os.Setenv("GEMINI_API_URL", server.URL)
	os.Unsetenv("GEMINI_FUNCTION_CALLING") // デフォルト: 有効

	p := NewGeminiProvider("test-key")
	history := []Message{{Role: "user", Content: "Read /test/file.txt"}}

	result, err := p.ChatWithTools(context.Background(), "System prompt", history, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// テキストとツール呼び出しが含まれていることを確認
	if result == "" {
		t.Error("ChatWithTools() should return non-empty result")
	}
}

func TestGeminiProvider_ChatWithTools_FunctionCallingDisabled(t *testing.T) {
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

		w.Header().Set("Content-Type", "application/json")
		responses := []GeminiResponse{
			{Candidates: []GeminiCandidate{{Content: GeminiContent{Parts: []GeminiPart{{Text: "Text mode response"}}}}}},
		}
		_ = json.NewEncoder(w).Encode(responses)
	})

	originalURL := os.Getenv("GEMINI_API_URL")
	originalFC := os.Getenv("GEMINI_FUNCTION_CALLING")
	defer func() {
		os.Setenv("GEMINI_API_URL", originalURL)
		os.Setenv("GEMINI_FUNCTION_CALLING", originalFC)
	}()
	os.Setenv("GEMINI_API_URL", server.URL)
	os.Setenv("GEMINI_FUNCTION_CALLING", "0") // 無効化

	p := NewGeminiProvider("test-key")
	history := []Message{{Role: "user", Content: "Hello"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Text mode response" {
		t.Errorf("ChatWithTools() = %q, want 'Text mode response'", result)
	}
}

func TestGeminiProvider_ChatWithTools_MCPEnabled(t *testing.T) {
	// MCP有効時はテキストモードにフォールバック
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		// MCP有効時は tools が含まれない
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if _, hasTools := req["tools"]; hasTools {
			t.Error("Request should NOT include tools when MCP is enabled")
		}

		w.Header().Set("Content-Type", "application/json")
		responses := []GeminiResponse{
			{Candidates: []GeminiCandidate{{Content: GeminiContent{Parts: []GeminiPart{{Text: "MCP text mode"}}}}}},
		}
		_ = json.NewEncoder(w).Encode(responses)
	})

	originalURL := os.Getenv("GEMINI_API_URL")
	originalFC := os.Getenv("GEMINI_FUNCTION_CALLING")
	defer func() {
		os.Setenv("GEMINI_API_URL", originalURL)
		os.Setenv("GEMINI_FUNCTION_CALLING", originalFC)
	}()
	os.Setenv("GEMINI_API_URL", server.URL)
	os.Unsetenv("GEMINI_FUNCTION_CALLING") // デフォルト: 有効

	p := NewGeminiProvider("test-key")
	p.SetMCPEnabled(true) // MCP有効化

	history := []Message{{Role: "user", Content: "Hello"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "MCP text mode" {
		t.Errorf("ChatWithTools() = %q, want 'MCP text mode'", result)
	}
}

func TestGetGeminiFunctionCallingURL(t *testing.T) {
	originalURL := os.Getenv("GEMINI_API_URL")
	defer os.Setenv("GEMINI_API_URL", originalURL)

	t.Run("DefaultURL", func(t *testing.T) {
		os.Unsetenv("GEMINI_API_URL")
		url := getGeminiFunctionCallingURL("gemini-2.0-flash-exp")
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

func TestHandleFunctionCallingResponse_Array(t *testing.T) {
	// 配列形式のレスポンス
	responses := []GeminiFunctionResponse{
		{
			Candidates: []GeminiFunctionCandidate{
				{
					Content: GeminiFunctionContent{
						Parts: []GeminiFunctionPart{
							{Text: "Here's the result: "},
							{FunctionCall: &GeminiFunctionCall{
								Name: "read_file",
								Args: map[string]any{"path": "/test.txt"},
							}},
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(responses)

	p := NewGeminiProvider("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}

	// テキストとツール呼び出しが含まれていることを確認
	if result == "" {
		t.Error("Result should not be empty")
	}
}

func TestHandleFunctionCallingResponse_SingleObject(t *testing.T) {
	// 単一オブジェクト形式のレスポンス
	response := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{FunctionCall: &GeminiFunctionCall{
							Name: "write_file",
							Args: map[string]any{"path": "/out.txt", "content": "hello"},
						}},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(response)

	p := NewGeminiProvider("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}

	if result == "" {
		t.Error("Result should not be empty")
	}
}
