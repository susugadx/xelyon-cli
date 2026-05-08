package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/kimi"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestExecuteWebSearch_UsesSearchProviderModelWhenProviderOverrideDiffers(t *testing.T) {
	resetWebSearchCacheForTest()

	query := "override-provider-model-" + t.Name()
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

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"output": [
				{
					"type": "message",
					"content": [
						{
							"type": "output_text",
							"text": "Provider override native web search succeeded.",
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
	t.Cleanup(func() {
		_ = os.Setenv("OPENAI_RESPONSES_URL", oldURL)
		_ = os.Setenv("OPENAI_API_KEY", oldKey)
	})
	_ = os.Setenv("OPENAI_RESPONSES_URL", server.URL)
	_ = os.Setenv("OPENAI_API_KEY", "test-key")

	cfg := config.DefaultConfig()
	cfg.WebSearch.Provider = "openai"
	cfg.ProviderModels["openai"] = config.ProviderModelConfig{
		DefaultModel:    "gpt-5.2-codex",
		MaxOutputTokens: 4096,
	}
	cfg.OpenAI.ResponsesAPIModels = append(cfg.OpenAI.ResponsesAPIModels, "gpt-5.2-codex")

	result := ExecuteWebSearch(tools.ExecutionContext{
		ProviderName: "deepseek",
		Model:        "deepseek-chat",
		Stdout:       io.Discard,
		Stderr:       io.Discard,
		Config:       cfg,
		AutoApprove:  true,
	}, query)

	if !strings.Contains(result, "Provider override native web search succeeded.") {
		t.Fatalf("result should contain native search summary, got %q", result)
	}

	var req map[string]any
	select {
	case req = <-requestCh:
	default:
		t.Fatal("expected native OpenAI web search request to be sent")
	}

	if got := req["model"]; got != "gpt-5.2-codex" {
		t.Fatalf("model = %#v, want %q", got, "gpt-5.2-codex")
	}
	if got := req["model"]; got == "deepseek-chat" {
		t.Fatal("search provider override must not reuse the main provider model")
	}
}

func TestExecuteWebSearch_KimiPreservesSessionPromptCacheScope(t *testing.T) {
	resetWebSearchCacheForTest()

	query := "kimi-prompt-cache-scope-" + t.Name()
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q, want Bearer test-key", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Kimi ExecuteWebSearch scope ok.\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2}}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	t.Setenv("KIMI_API_URL", server.URL)
	t.Setenv("MOONSHOT_API_KEY", "test-key")

	cfg := config.DefaultConfig()
	cfg.WebSearch.Provider = "kimi"
	cfg.SetProviderModelConfig("kimi", config.ProviderModelConfig{
		DefaultModel:    "kimi-search-test",
		MaxOutputTokens: 123,
	})
	requestCtx := api.WithPromptCacheScope(context.Background(), api.PromptCacheScope{SessionID: "execute-web-search-session"})

	result := ExecuteWebSearch(tools.ExecutionContext{
		Context:           requestCtx,
		ProviderName:      "kimi",
		ProviderConfigKey: "kimi",
		Model:             "kimi-search-test",
		Stdout:            io.Discard,
		Stderr:            io.Discard,
		Config:            cfg,
		AutoApprove:       true,
	}, query)

	if !strings.Contains(result, "Kimi ExecuteWebSearch scope ok.") {
		t.Fatalf("result should contain native search summary, got %q", result)
	}
	if captured == nil {
		t.Fatal("expected native Kimi web search request to be sent")
	}
	key, ok := captured["prompt_cache_key"].(string)
	if !ok || key == "" {
		t.Fatalf("prompt_cache_key = %#v, want non-empty string", captured["prompt_cache_key"])
	}
	if !strings.HasPrefix(key, "xelyon:kimi:v1:") {
		t.Fatalf("prompt_cache_key = %q, want session-aware Kimi key", key)
	}
	if strings.HasPrefix(key, "xelyon:v2:") {
		t.Fatalf("prompt_cache_key = %q, want Kimi session scope key instead of project prompt fallback", key)
	}
}
