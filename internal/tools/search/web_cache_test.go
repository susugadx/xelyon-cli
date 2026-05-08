package search

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	_ "github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

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

func TestSearchWithCache_DoesNotShareAcrossClaudeAliasOwners(t *testing.T) {
	resetWebSearchCacheForTest()

	query := "shared-claude-alias-cache-" + t.Name()
	model := "claude-sonnet-4-6"
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"cached claude alias result"}]}`))
	}))
	defer server.Close()

	oldURL := os.Getenv("ANTHROPIC_API_URL")
	oldKey := os.Getenv("ANTHROPIC_API_KEY")
	t.Cleanup(func() {
		_ = os.Setenv("ANTHROPIC_API_URL", oldURL)
		_ = os.Setenv("ANTHROPIC_API_KEY", oldKey)
	})
	_ = os.Setenv("ANTHROPIC_API_URL", server.URL)
	_ = os.Setenv("ANTHROPIC_API_KEY", "test-key")

	cfg := config.DefaultConfig()
	ctx := tools.WithConfig(context.Background(), cfg)

	result, cached, err := searchWithCache(ctx, cfg, "anthropic", query, model)
	if err != nil {
		t.Fatalf("anthropic first search failed: %v", err)
	}
	if cached {
		t.Fatal("anthropic first search should not be cached")
	}
	if !strings.Contains(result, "cached claude alias result") {
		t.Fatalf("result = %q, want cached response text", result)
	}

	_, cached, err = searchWithCache(ctx, cfg, "claude", query, model)
	if err != nil {
		t.Fatalf("claude second search failed: %v", err)
	}
	if cached {
		t.Fatal("claude second search must not reuse the anthropic cache bucket")
	}
	if callCount != 2 {
		t.Fatalf("Claude native web search was called %d times, want 2 separate alias-owner requests", callCount)
	}
}

func TestSearchWithCache_SharesWithinSameExactOwner(t *testing.T) {
	resetWebSearchCacheForTest()

	query := "same-owner-cache-" + t.Name()
	model := "claude-sonnet-4-6"
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"same owner cached result"}]}`))
	}))
	defer server.Close()

	oldURL := os.Getenv("ANTHROPIC_API_URL")
	oldKey := os.Getenv("ANTHROPIC_API_KEY")
	t.Cleanup(func() {
		_ = os.Setenv("ANTHROPIC_API_URL", oldURL)
		_ = os.Setenv("ANTHROPIC_API_KEY", oldKey)
	})
	_ = os.Setenv("ANTHROPIC_API_URL", server.URL)
	_ = os.Setenv("ANTHROPIC_API_KEY", "test-key")

	cfg := config.DefaultConfig()
	ctx := tools.WithConfig(context.Background(), cfg)

	if _, cached, err := searchWithCache(ctx, cfg, "anthropic", query, model); err != nil {
		t.Fatalf("anthropic first search failed: %v", err)
	} else if cached {
		t.Fatal("anthropic first search should not be cached")
	}
	if _, cached, err := searchWithCache(ctx, cfg, "anthropic", query, model); err != nil {
		t.Fatalf("anthropic second search failed: %v", err)
	} else if !cached {
		t.Fatal("anthropic second search should hit cache")
	}
	if _, cached, err := searchWithCache(ctx, cfg, "claude", query, model); err != nil {
		t.Fatalf("claude first search failed: %v", err)
	} else if cached {
		t.Fatal("claude first search should not reuse anthropic owner cache")
	}
	if _, cached, err := searchWithCache(ctx, cfg, "claude", query, model); err != nil {
		t.Fatalf("claude second search failed: %v", err)
	} else if !cached {
		t.Fatal("claude second search should hit cache")
	}
	if callCount != 2 {
		t.Fatalf("Claude native web search was called %d times, want 2 calls (one per exact owner)", callCount)
	}
}

func TestSearchWithCache_DoesNotShareAcrossDifferentRuntimeProviders(t *testing.T) {
	resetWebSearchCacheForTest()

	query := "different-runtime-cache-" + t.Name()
	anthropicCalls := 0
	openAICalls := 0

	anthropicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		anthropicCalls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"anthropic result"}]}`))
	}))
	defer anthropicServer.Close()

	openAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		openAICalls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"output": [
				{
					"type": "message",
					"content": [
						{"type": "output_text", "text": "openai result"}
					]
				}
			]
		}`))
	}))
	defer openAIServer.Close()

	oldAnthropicURL := os.Getenv("ANTHROPIC_API_URL")
	oldAnthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	oldOpenAIURL := os.Getenv("OPENAI_RESPONSES_URL")
	oldOpenAIKey := os.Getenv("OPENAI_API_KEY")
	t.Cleanup(func() {
		_ = os.Setenv("ANTHROPIC_API_URL", oldAnthropicURL)
		_ = os.Setenv("ANTHROPIC_API_KEY", oldAnthropicKey)
		_ = os.Setenv("OPENAI_RESPONSES_URL", oldOpenAIURL)
		_ = os.Setenv("OPENAI_API_KEY", oldOpenAIKey)
	})
	_ = os.Setenv("ANTHROPIC_API_URL", anthropicServer.URL)
	_ = os.Setenv("ANTHROPIC_API_KEY", "anthropic-test-key")
	_ = os.Setenv("OPENAI_RESPONSES_URL", openAIServer.URL)
	_ = os.Setenv("OPENAI_API_KEY", "openai-test-key")

	cfg := config.DefaultConfig()
	cfg.OpenAI.ResponsesAPIModels = append(cfg.OpenAI.ResponsesAPIModels, "gpt-4o")
	ctx := tools.WithConfig(context.Background(), cfg)

	if _, cached, err := searchWithCache(ctx, cfg, "anthropic", query, "claude-sonnet-4-6"); err != nil {
		t.Fatalf("anthropic search failed: %v", err)
	} else if cached {
		t.Fatal("anthropic first search should not be cached")
	}

	if _, cached, err := searchWithCache(ctx, cfg, "openai", query, "gpt-4o"); err != nil {
		t.Fatalf("openai search failed: %v", err)
	} else if cached {
		t.Fatal("openai search must not reuse anthropic cache bucket")
	}

	if anthropicCalls != 1 {
		t.Fatalf("anthropic native web search was called %d times, want 1", anthropicCalls)
	}
	if openAICalls != 1 {
		t.Fatalf("openai native web search was called %d times, want 1", openAICalls)
	}
}
