package claude

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestWebSearch_Success(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("x-api-key = %q, want test-key", got)
		}
		if got := r.Header.Get("anthropic-beta"); !strings.Contains(got, webSearchBetaHeader) {
			t.Fatalf("anthropic-beta = %q, want %q", got, webSearchBetaHeader)
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		tools, ok := req["tools"].([]any)
		if !ok || len(tools) != 1 {
			t.Fatalf("tools = %#v, want 1 tool", req["tools"])
		}
		tool, ok := tools[0].(map[string]any)
		if !ok || tool["type"] != "web_search_20250305" {
			t.Fatalf("tool = %#v, want type=web_search_20250305", tools[0])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"content": [
				{
					"type": "text",
					"text": "Anthropic shipped a web search tool.",
					"citations": [
						{"type": "web_search_result_location", "title": "Anthropic Docs", "url": "https://docs.anthropic.com/en/docs/build-with-claude/tool-use/web-search-tool"}
					]
				}
			]
		}`))
	})

	oldURL := os.Getenv("ANTHROPIC_API_URL")
	oldKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_URL", server.URL)
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer os.Setenv("ANTHROPIC_API_URL", oldURL)
	defer os.Setenv("ANTHROPIC_API_KEY", oldKey)

	result, err := WebSearch("anthropic web search tool", "claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("WebSearch() error = %v", err)
	}
	if !strings.Contains(result, "Summary:") {
		t.Fatalf("result should contain Summary, got %q", result)
	}
	if !strings.Contains(result, "https://docs.anthropic.com/en/docs/build-with-claude/tool-use/web-search-tool") {
		t.Fatalf("result should contain citation URL, got %q", result)
	}
}
