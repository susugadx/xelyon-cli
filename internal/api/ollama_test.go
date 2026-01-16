package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestNewOllamaProvider(t *testing.T) {
	baseURL := "http://localhost:11434"
	provider := NewOllamaProvider(baseURL)

	if provider == nil {
		t.Fatal("NewOllamaProvider() returned nil")
	}
}

func TestOllamaProvider_Name(t *testing.T) {
	provider := NewOllamaProvider("http://localhost:11434")

	name := provider.Name()
	if name != "Ollama" {
		t.Errorf("Name() = %v, want 'Ollama'", name)
	}
}

func TestOllamaProvider_SupportsImages(t *testing.T) {
	provider := NewOllamaProvider("http://localhost:11434")

	supports := provider.SupportsImages()
	if supports {
		t.Error("SupportsImages() = true, want false (Ollama does not support images)")
	}
}

func TestNewOllamaProvider_DefaultURL(t *testing.T) {
	p := NewOllamaProvider("")
	if p.baseURL != "http://localhost:11434" {
		t.Errorf("baseURL = %q, want 'http://localhost:11434'", p.baseURL)
	}
}

func TestNewOllamaProvider_CustomURL(t *testing.T) {
	customURL := "http://custom-ollama:8080"
	p := NewOllamaProvider(customURL)
	if p.baseURL != customURL {
		t.Errorf("baseURL = %q, want %q", p.baseURL, customURL)
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

func TestOllamaProvider_ChatWithTools_Streaming(t *testing.T) {
	server := mockAPIServer(t, ollamaStreamingHandler([]string{"Hello", " from", " Ollama"}))

	p := NewOllamaProvider(server.URL)
	history := []Message{{Role: "user", Content: "Hi"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "llama3")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Hello from Ollama" {
		t.Errorf("ChatWithTools() = %q, want 'Hello from Ollama'", result)
	}
}

func TestOllamaProvider_ChatWithTools_APIError(t *testing.T) {
	server := mockAPIServer(t, errorHandler(500, "Internal Server Error"))

	p := NewOllamaProvider(server.URL)
	history := []Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for 500 status")
	}
}

func TestOllamaProvider_ChatWithTools_RateLimit(t *testing.T) {
	server := mockAPIServer(t, rateLimitHandler("60"))

	p := NewOllamaProvider(server.URL)
	history := []Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for rate limit")
	}
}

func TestOllamaProvider_ChatWithImage(t *testing.T) {
	server := mockAPIServer(t, ollamaStreamingHandler([]string{"Image", " ignored"}))

	p := NewOllamaProvider(server.URL)
	image := &ImageData{Base64: "dGVzdA==", MediaType: "image/png"}

	// Ollamaは画像非対応なのでテキストのみ送信
	result, err := p.ChatWithImage(context.Background(), "System", nil, "Describe this", image, "")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "Image ignored" {
		t.Errorf("ChatWithImage() = %q, want 'Image ignored'", result)
	}
}

func TestOllamaProvider_ListModels_Success(t *testing.T) {
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

	p := NewOllamaProvider(server.URL)
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

func TestOllamaProvider_ListModels_APIError(t *testing.T) {
	server := mockAPIServer(t, errorHandler(500, "Internal Server Error"))

	p := NewOllamaProvider(server.URL)
	_, err := p.ListModels()
	if err == nil {
		t.Error("ListModels() should return error for 500 status")
	}
}
