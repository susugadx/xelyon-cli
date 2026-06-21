package search

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/kimi"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestExecuteWebSearch_UsesKimiNativeSearchProviderAndUsageCallback(t *testing.T) {
	resetWebSearchCacheForTest()

	query := "kimi-native-web-search-" + t.Name()
	requestCh := make(chan map[string]any, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q, want Bearer test-key", got)
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		requestCh <- req

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Kimi native web search succeeded.\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":13,\"completion_tokens\":4,\"cached_tokens\":2}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	oldURL := os.Getenv("KIMI_API_URL")
	oldKey := os.Getenv("MOONSHOT_API_KEY")
	t.Cleanup(func() {
		_ = os.Setenv("KIMI_API_URL", oldURL)
		_ = os.Setenv("MOONSHOT_API_KEY", oldKey)
	})
	_ = os.Setenv("KIMI_API_URL", server.URL)
	_ = os.Setenv("MOONSHOT_API_KEY", "test-key")

	cfg := config.DefaultConfig()
	cfg.WebSearch.Provider = "kimi"
	cfg.SetProviderModelConfig("kimi", config.ProviderModelConfig{
		DefaultModel:    "kimi-web-search-default",
		MaxOutputTokens: 88,
	})

	var gotUsage api.Usage
	var gotUsageProvider string
	var gotUsageModel string
	result := ExecuteWebSearch(tools.ExecutionContext{
		ProviderName: "deepseek",
		Model:        "deepseek-v4-flash",
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

	if !strings.Contains(result, "Kimi native web search succeeded.") {
		t.Fatalf("result should contain native Kimi search summary, got %q", result)
	}

	var req map[string]any
	select {
	case req = <-requestCh:
	default:
		t.Fatal("expected native Kimi web search request to be sent")
	}
	if got := req["model"]; got != "kimi-web-search-default" {
		t.Fatalf("model = %#v, want Kimi configured default model", got)
	}
	if req["max_completion_tokens"] != float64(88) {
		t.Fatalf("max_completion_tokens = %#v, want 88", req["max_completion_tokens"])
	}
	if _, ok := req["max_tokens"]; ok {
		t.Fatal("max_tokens should be omitted")
	}
	thinking, ok := req["thinking"].(map[string]any)
	if !ok || thinking["type"] != "disabled" {
		t.Fatalf("thinking = %#v, want disabled", req["thinking"])
	}
	toolsPayload, ok := req["tools"].([]any)
	if !ok || len(toolsPayload) != 1 {
		t.Fatalf("tools = %#v, want Kimi builtin web_search tool", req["tools"])
	}
	tool, ok := toolsPayload[0].(map[string]any)
	if !ok || tool["type"] != "builtin_function" {
		t.Fatalf("tool = %#v, want builtin_function", toolsPayload[0])
	}
	function, ok := tool["function"].(map[string]any)
	if !ok || function["name"] != "$web_search" {
		t.Fatalf("tool.function = %#v, want $web_search", tool["function"])
	}
	if _, ok := req["tool_choice"]; ok {
		t.Fatalf("tool_choice = %#v, want omitted", req["tool_choice"])
	}
	if gotUsage.InputTokens != 13 || gotUsage.OutputTokens != 4 || gotUsage.CachedInputTokens != 2 {
		t.Fatalf("usage = %+v, want input=13 output=4 cached=2", gotUsage)
	}
	if gotUsageProvider != "kimi" || gotUsageModel != "kimi-web-search-default" {
		t.Fatalf("usage owner = %s/%s, want kimi/kimi-web-search-default", gotUsageProvider, gotUsageModel)
	}
}
