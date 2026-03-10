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

	_ "github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestResolveSearchProvider(t *testing.T) {
	tests := []struct {
		configProvider string
		mainProvider   string
		want           string
	}{
		// config設定あり → そちらを使用
		{"gemini", "deepseek", "gemini"},
		{"openai", "deepseek", "openai"},
		{"claude", "openai", "claude"},

		// config未設定 → メインプロバイダー
		{"", "openai", "openai"},
		{"", "gemini", "gemini"},
		{"", "claude", "claude"},
		{"", "anthropic", "claude"},

		// config未設定 + ネイティブ非対応 → 空文字
		{"", "deepseek", ""},
		{"", "openrouter", ""},
		{"", "groq", ""},
		{"", "ollama", ""},

		// 無効なconfig設定 → メインにフォールバック
		{"invalid", "openai", "openai"},
		{"invalid", "deepseek", ""},
	}

	for _, tt := range tests {
		cfg := &config.Config{WebSearch: config.WebSearchConfig{Provider: tt.configProvider}}
		got := resolveSearchProvider(cfg, tt.mainProvider)
		if got != tt.want {
			t.Errorf("resolveSearchProvider(config=%q, main=%q) = %q, want %q", tt.configProvider, tt.mainProvider, got, tt.want)
		}
	}
}

func TestWebSearchProviderError(t *testing.T) {
	msg := webSearchProviderError()
	if !strings.Contains(msg, "web_search.provider") {
		t.Fatal("error message should mention web_search.provider")
	}
	if !strings.Contains(msg, "gemini") {
		t.Fatal("error message should mention gemini as an option")
	}
}

func TestSearchWithCache_ProviderScopedKeys(t *testing.T) {
	resetWebSearchCacheForTest()

	cfg := config.DefaultConfig()
	providerA := strings.ToLower(t.Name()) + "-openai"
	providerB := strings.ToLower(t.Name()) + "-claude"
	query := "same query"

	calls := map[string]int{}
	websearch.RegisterWithContext(providerA, func(ctx context.Context, q, model string) (string, error) {
		calls[providerA]++
		return fmt.Sprintf("%s:%s", providerA, q), nil
	})
	websearch.RegisterWithContext(providerB, func(ctx context.Context, q, model string) (string, error) {
		calls[providerB]++
		return fmt.Sprintf("%s:%s", providerB, q), nil
	})

	result, cached, err := searchWithCache(context.Background(), cfg, providerA, query, "")
	if err != nil {
		t.Fatalf("providerA first search failed: %v", err)
	}
	if cached {
		t.Fatal("providerA first search should not be cached")
	}
	if result == "" {
		t.Fatal("providerA first search should return a result")
	}

	_, cached, err = searchWithCache(context.Background(), cfg, providerA, query, "")
	if err != nil {
		t.Fatalf("providerA second search failed: %v", err)
	}
	if !cached {
		t.Fatal("providerA second search should be cached")
	}

	_, cached, err = searchWithCache(context.Background(), cfg, providerB, query, "")
	if err != nil {
		t.Fatalf("providerB first search failed: %v", err)
	}
	if cached {
		t.Fatal("providerB first search should not be cached because cache key is provider-scoped")
	}

	if calls[providerA] != 1 {
		t.Fatalf("providerA executor called %d times, want 1", calls[providerA])
	}
	if calls[providerB] != 1 {
		t.Fatalf("providerB executor called %d times, want 1", calls[providerB])
	}
}

func TestSearchWithCache_CacheDisabled(t *testing.T) {
	resetWebSearchCacheForTest()

	cfg := config.DefaultConfig()
	cfg.WebSearch.CacheEnabled = false
	provider := strings.ToLower(t.Name())
	calls := 0
	websearch.RegisterWithContext(provider, func(ctx context.Context, q, model string) (string, error) {
		calls++
		return "ok", nil
	})

	_, cached, err := searchWithCache(context.Background(), cfg, provider, "disabled-test", "")
	if err != nil {
		t.Fatalf("first search failed: %v", err)
	}
	if cached {
		t.Fatal("first search should not be cached when cache is disabled")
	}

	_, cached, err = searchWithCache(context.Background(), cfg, provider, "disabled-test", "")
	if err != nil {
		t.Fatalf("second search failed: %v", err)
	}
	if cached {
		t.Fatal("second search should not be cached when cache is disabled")
	}

	if calls != 2 {
		t.Fatalf("executor called %d times, want 2", calls)
	}
}

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

func resetWebSearchCacheForTest() {
	webSearchCacheMu.Lock()
	defer webSearchCacheMu.Unlock()
	webSearchCache = nil
	webSearchCacheSettings = webSearchCacheConfig{}
}
