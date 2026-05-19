package search

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/gemini"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestExecuteWebSearch_GeminiReportsUsageAttribution(t *testing.T) {
	resetWebSearchCacheForTest()

	query := "gemini-usage-attribution-" + t.Name()
	requestCh := make(chan map[string]any, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("x-goog-api-key"); got != "test-key" {
			t.Fatalf("x-goog-api-key = %q, want test-key", got)
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		requestCh <- req

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates": [
				{
					"content": {"parts": [{"text": "Gemini native web search succeeded."}]},
					"groundingMetadata": {
						"groundingChunks": [
							{"web": {"uri": "https://example.com/gemini", "title": "Gemini Example"}}
						]
					}
				}
			],
			"usageMetadata": {
				"promptTokenCount": 17,
				"candidatesTokenCount": 5,
				"thoughtsTokenCount": 3,
				"cachedContentTokenCount": 4
			}
		}`))
	}))
	defer server.Close()

	t.Setenv("GEMINI_API_URL", server.URL)
	t.Setenv("GEMINI_API_KEY", "test-key")

	cfg := config.DefaultConfig()
	cfg.WebSearch.Provider = "gemini"
	cfg.SetProviderModelConfig("gemini", config.ProviderModelConfig{
		DefaultModel: "gemini-3.1-pro-preview-customtools",
	})

	var gotUsage api.Usage
	var gotUsageProvider string
	var gotUsageModel string
	result := ExecuteWebSearch(tools.ExecutionContext{
		ProviderName: "deepseek",
		Model:        "deepseek-chat",
		Stdout:       io.Discard,
		Stderr:       io.Discard,
		Config:       cfg,
		AutoApprove:  true,
		UsageAttribution: func(provider, model string, usage api.Usage) {
			gotUsageProvider = provider
			gotUsageModel = model
			gotUsage = usage
		},
	}, query)

	if !strings.Contains(result, "Gemini native web search succeeded.") {
		t.Fatalf("result should contain native Gemini search summary, got %q", result)
	}

	var req map[string]any
	select {
	case req = <-requestCh:
	default:
		t.Fatal("expected native Gemini web search request to be sent")
	}
	toolsPayload, ok := req["tools"].([]any)
	if !ok || len(toolsPayload) != 1 {
		t.Fatalf("tools = %#v, want Gemini native web search tool", req["tools"])
	}
	tool, ok := toolsPayload[0].(map[string]any)
	if !ok || tool["google_search"] == nil {
		t.Fatalf("tool = %#v, want google_search", toolsPayload[0])
	}
	if gotUsage.InputTokens != 17 || gotUsage.OutputTokens != 5 || gotUsage.ThinkingTokens != 3 || gotUsage.CachedInputTokens != 4 {
		t.Fatalf("usage = %+v, want input=17 output=5 thinking=3 cached=4", gotUsage)
	}
	if gotUsageProvider != "gemini" || gotUsageModel != "gemini-3.1-pro-preview-customtools" {
		t.Fatalf("usage owner = %s/%s, want gemini/gemini-3.1-pro-preview-customtools", gotUsageProvider, gotUsageModel)
	}
}
