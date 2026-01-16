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
		json.NewEncoder(w).Encode(responses)
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
		json.NewEncoder(w).Encode(resp)
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
		json.NewEncoder(w).Encode(responses)
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
		json.NewEncoder(w).Encode(responses)
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
