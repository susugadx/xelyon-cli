package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestWebSearch_Success(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequestMethod(t, r, "POST")
		assertJSONContentType(t, r)
		assertRequestHeader(t, r, "x-goog-api-key", "test-key")

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		tools, ok := req["tools"].([]any)
		if !ok || len(tools) != 1 {
			t.Fatalf("tools = %#v, want 1 tool", req["tools"])
		}
		tool, ok := tools[0].(map[string]any)
		if !ok || tool["google_search"] == nil {
			t.Fatalf("tool = %#v, want google_search", tools[0])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates": [
				{
					"content": {"parts": [{"text": "Go 1.24 added new language and toolchain improvements."}]},
					"groundingMetadata": {
						"groundingChunks": [
							{"web": {"uri": "https://go.dev/doc/go1.24", "title": "Go 1.24 Release Notes"}},
							{"web": {"uri": "https://go.dev/blog", "title": "Go Blog"}}
						]
					}
				}
			]
		}`))
	})

	oldURL := os.Getenv("GEMINI_API_URL")
	oldKey := os.Getenv("GEMINI_API_KEY")
	os.Setenv("GEMINI_API_URL", server.URL)
	os.Setenv("GEMINI_API_KEY", "test-key")
	defer os.Setenv("GEMINI_API_URL", oldURL)
	defer os.Setenv("GEMINI_API_KEY", oldKey)

	ctx := config.WithContext(context.Background(), config.DefaultConfig())
	result, err := WebSearchWithContext(ctx, "go 1.24 release", "gemini-3.1-pro-preview-customtools")
	if err != nil {
		t.Fatalf("WebSearchWithContext() error = %v", err)
	}
	if !strings.Contains(result, "Summary:") {
		t.Fatalf("result should contain Summary, got %q", result)
	}
	if !strings.Contains(result, "https://go.dev/doc/go1.24") {
		t.Fatalf("result should contain grounding source, got %q", result)
	}
}

func TestWebSearch_LegacyModelUsesGoogleSearchRetrieval(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		tools, ok := req["tools"].([]any)
		if !ok || len(tools) != 1 {
			t.Fatalf("tools = %#v, want 1 tool", req["tools"])
		}
		tool, ok := tools[0].(map[string]any)
		if !ok || tool["google_search_retrieval"] == nil {
			t.Fatalf("tool = %#v, want google_search_retrieval", tools[0])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"Legacy grounded response."}]}}]}`))
	})

	oldURL := os.Getenv("GEMINI_API_URL")
	oldKey := os.Getenv("GEMINI_API_KEY")
	os.Setenv("GEMINI_API_URL", server.URL)
	os.Setenv("GEMINI_API_KEY", "test-key")
	defer os.Setenv("GEMINI_API_URL", oldURL)
	defer os.Setenv("GEMINI_API_KEY", oldKey)

	ctx := config.WithContext(context.Background(), config.DefaultConfig())
	result, err := WebSearchWithContext(ctx, "legacy grounding", "gemini-1.5-pro")
	if err != nil {
		t.Fatalf("WebSearchWithContext() error = %v", err)
	}
	if !strings.Contains(result, "Legacy grounded response.") {
		t.Fatalf("result should contain model output, got %q", result)
	}
}
