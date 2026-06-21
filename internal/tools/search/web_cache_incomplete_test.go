package search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	_ "github.com/susugadx/xelyon-cli/internal/api/providers/kimi"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestSearchWithCache_DoesNotCacheIncompleteKimiWebSearch(t *testing.T) {
	resetWebSearchCacheForTest()

	query := "kimi-incomplete-web-search-cache-" + t.Name()
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		w.Header().Set("Content-Type", "text/event-stream")
		if requestCount == 1 {
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"partial Kimi native web search result\"}}]}\n\n"))
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"length\"}],\"usage\":{\"prompt_tokens\":13,\"completion_tokens\":4}}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Complete Kimi native web search result.\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":13,\"completion_tokens\":4}}\n\n"))
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
	cfg.WebSearch.CacheEnabled = true
	cfg.WebSearch.CacheSize = 50
	cfg.WebSearch.Provider = "kimi"
	cfg.SetProviderModelConfig("kimi", config.ProviderModelConfig{
		DefaultModel:    "kimi-k2.6",
		MaxOutputTokens: 88,
	})
	ctx := config.WithContext(context.Background(), cfg)

	_, cached, err := searchWithCache(ctx, cfg, "kimi", query, "kimi-k2.6")
	if err == nil {
		t.Fatal("first searchWithCache() error = nil, want incomplete finish_reason error")
	}
	if cached {
		t.Fatal("first searchWithCache() cached = true, want false for failed response")
	}

	result, cached, err := searchWithCache(ctx, cfg, "kimi", query, "kimi-k2.6")
	if err != nil {
		t.Fatalf("second searchWithCache() error = %v", err)
	}
	if cached {
		t.Fatal("second searchWithCache() cached = true, want live request after failed response")
	}
	if !strings.Contains(result, "Complete Kimi native web search result.") {
		t.Fatalf("second searchWithCache() = %q, want complete result", result)
	}
	if requestCount != 2 {
		t.Fatalf("requestCount = %d, want 2 live requests", requestCount)
	}
}
