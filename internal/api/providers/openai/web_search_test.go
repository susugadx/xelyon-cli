package openai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestWebSearch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", got)
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
		if !ok || tool["type"] != "web_search" {
			t.Fatalf("tool = %#v, want type=web_search", tools[0])
		}

		include, ok := req["include"].([]any)
		if !ok || len(include) != 1 || include[0] != "web_search_call.action.sources" {
			t.Fatalf("include = %#v, want web_search_call.action.sources", req["include"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"output": [
				{
					"type": "web_search_call",
					"action": {
						"sources": [
							{"type":"url","title":"OpenAI Blog","url":"https://openai.com/blog"}
						]
					}
				},
				{
					"type": "message",
					"content": [
						{
							"type": "output_text",
							"text": "OpenAI released a new feature.",
							"annotations": [
								{"type":"url_citation","title":"OpenAI Blog","url":"https://openai.com/blog"}
							]
						}
					]
				}
			]
		}`))
	}))
	defer server.Close()

	oldURL := os.Getenv("OPENAI_RESPONSES_URL")
	oldKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_RESPONSES_URL", server.URL)
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Setenv("OPENAI_RESPONSES_URL", oldURL)
	defer os.Setenv("OPENAI_API_KEY", oldKey)

	result, err := WebSearch("latest openai news", "gpt-4o")
	if err != nil {
		t.Fatalf("WebSearch() error = %v", err)
	}

	if !strings.Contains(result, "Summary:") {
		t.Fatalf("result should contain Summary, got %q", result)
	}
	if !strings.Contains(result, "OpenAI released a new feature.") {
		t.Fatalf("result should contain summary text, got %q", result)
	}
	if strings.Count(result, "https://openai.com/blog") != 1 {
		t.Fatalf("result should dedupe sources, got %q", result)
	}
}

func TestWebSearch_UnsupportedModel(t *testing.T) {
	oldKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Setenv("OPENAI_API_KEY", oldKey)

	_, err := WebSearch("latest openai news", "gpt-4")
	if err == nil {
		t.Fatal("WebSearch() should fail for non-Responses models")
	}
	if !strings.Contains(err.Error(), "does not support Responses API web search") {
		t.Fatalf("unexpected error: %v", err)
	}
}
