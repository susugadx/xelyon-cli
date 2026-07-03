package search

import (
	"context"
	"fmt"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestSearchWebUsesConfiguredProviderAndParsesResults(t *testing.T) {
	resetWebSearchCacheForTest()
	provider := "gemini"
	query := "OpenAI API web_search documentation"
	calls := 0
	websearch.RegisterWithContextForTest(t, provider, func(_ context.Context, gotQuery, model string) (string, error) {
		calls++
		if gotQuery != query {
			t.Fatalf("query = %q, want %q", gotQuery, query)
		}
		if model != "gemini-test-model" {
			t.Fatalf("model = %q, want configured provider model", model)
		}
		return "Summary:\nresult\n\nSources:\n\n1. Search docs\n   URL: https://docs.example.test/search\n\n2. Other\n   URL: https://docs.example.test/other", nil
	})
	cfg := config.DefaultConfig()
	cfg.WebSearch.Provider = provider
	cfg.SetProviderModelConfig(provider, config.ProviderModelConfig{DefaultModel: "gemini-test-model"})

	got, err := SearchWeb(context.Background(), WebSearchRequest{
		Config:       cfg,
		MainProvider: "deepseek",
		Query:        query,
		MaxResults:   1,
	})
	if err != nil {
		t.Fatalf("SearchWeb() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	if got.Provider != provider {
		t.Fatalf("Provider = %q, want %q", got.Provider, provider)
	}
	if len(got.Results) != 1 {
		t.Fatalf("Results = %#v, want one bounded result", got.Results)
	}
	if !got.ResultsTruncated {
		t.Fatal("ResultsTruncated = false, want true when parsed results exceed MaxResults")
	}
	if got.Results[0].URL != "https://docs.example.test/search" || got.Results[0].SourceDomain != "docs.example.test" {
		t.Fatalf("result = %#v, want parsed URL/domain", got.Results[0])
	}
}

func TestSearchWebReportsResolvedUsageAttributionAndLegacyCallback(t *testing.T) {
	resetWebSearchCacheForTest()
	provider := "gemini"
	query := "usage attribution query"
	websearch.RegisterWithContextForTest(t, provider, func(ctx context.Context, gotQuery, model string) (string, error) {
		if gotQuery != query {
			t.Fatalf("query = %q, want %q", gotQuery, query)
		}
		if model != "gemini-usage-model" {
			t.Fatalf("model = %q, want configured provider model", model)
		}
		callback := websearch.UsageCallbackFromContext(ctx)
		if callback == nil {
			t.Fatal("UsageCallbackFromContext() = nil, want callback")
		}
		callback(api.Usage{InputTokens: 17, OutputTokens: 5, CachedInputTokens: 3})
		return "1. Docs\n   URL: https://docs.example.test/usage", nil
	})
	cfg := config.DefaultConfig()
	cfg.WebSearch.Provider = provider
	cfg.SetProviderModelConfig(provider, config.ProviderModelConfig{DefaultModel: "gemini-usage-model"})

	var legacyUsage api.Usage
	var attributedUsage api.Usage
	var attributedProvider string
	var attributedModel string
	_, err := SearchWeb(context.Background(), WebSearchRequest{
		Config:       cfg,
		MainProvider: "deepseek",
		MainModel:    "deepseek-v4",
		Query:        query,
		UsageCallback: func(usage api.Usage) {
			legacyUsage = usage
		},
		UsageAttribution: func(provider, model string, usage api.Usage) {
			attributedProvider = provider
			attributedModel = model
			attributedUsage = usage
		},
	})
	if err != nil {
		t.Fatalf("SearchWeb() error = %v", err)
	}
	if legacyUsage.InputTokens != 17 || legacyUsage.OutputTokens != 5 || legacyUsage.CachedInputTokens != 3 {
		t.Fatalf("legacy usage = %+v, want provider usage", legacyUsage)
	}
	if attributedProvider != provider || attributedModel != "gemini-usage-model" {
		t.Fatalf("usage owner = %s/%s, want gemini/gemini-usage-model", attributedProvider, attributedModel)
	}
	if attributedUsage.InputTokens != 17 || attributedUsage.OutputTokens != 5 || attributedUsage.CachedInputTokens != 3 {
		t.Fatalf("attributed usage = %+v, want provider usage", attributedUsage)
	}
}

func TestSearchWebPassesThinkingConfigToNativeProviderContext(t *testing.T) {
	resetWebSearchCacheForTest()
	query := "thinking inheritance query"
	websearch.RegisterWithContextForTest(t, "openai_subscription", func(ctx context.Context, gotQuery, model string) (string, error) {
		if gotQuery != query {
			t.Fatalf("query = %q, want %q", gotQuery, query)
		}
		if model != "gpt-5.5" {
			t.Fatalf("model = %q, want configured subscription model", model)
		}
		cfg := config.FromContext(ctx)
		if !cfg.Thinking.Enabled || cfg.Thinking.Level != "high" || !api.IsThinkingEnabled(ctx) {
			t.Fatalf("thinking config = %+v active=%t, want high enabled", cfg.Thinking, api.IsThinkingEnabled(ctx))
		}
		return "Summary:\nthinking inherited\n\nSources:\n\n1. Docs\n   URL: https://docs.example.test/thinking", nil
	})
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "high"
	cfg.WebSearch.Provider = "openai_subscription"
	cfg.SetProviderModelConfig("openai_subscription", config.ProviderModelConfig{DefaultModel: "gpt-5.5"})

	got, err := SearchWeb(context.Background(), WebSearchRequest{
		Config:       cfg,
		MainProvider: "deepseek",
		Query:        query,
	})
	if err != nil {
		t.Fatalf("SearchWeb() error = %v", err)
	}
	if got.Provider != "openai_subscription" || got.Model != "gpt-5.5" {
		t.Fatalf("provider/model = %s/%s, want openai_subscription/gpt-5.5", got.Provider, got.Model)
	}
}

func TestSearchWebCanonicalizesOpenAISubscriptionAliasBeforeUsageAndCacheOwners(t *testing.T) {
	resetWebSearchCacheForTest()
	query := "subscription alias owner query"
	calls := 0
	websearch.RegisterWithContextForTest(t, "openai_subscription", func(ctx context.Context, gotQuery, model string) (string, error) {
		calls++
		if gotQuery != query {
			t.Fatalf("query = %q, want %q", gotQuery, query)
		}
		if model != "gpt-5.5" {
			t.Fatalf("model = %q, want configured subscription model", model)
		}
		callback := websearch.UsageCallbackFromContext(ctx)
		if callback == nil {
			t.Fatal("UsageCallbackFromContext() = nil, want callback")
		}
		callback(api.Usage{InputTokens: 31, OutputTokens: 9})
		return "Summary:\nsubscription result\n\nSources:\n\n1. Docs\n   URL: https://docs.example.test/subscription", nil
	})
	cfg := config.DefaultConfig()
	cfg.WebSearch.Provider = "chatgpt"
	cfg.WebSearch.CacheEnabled = true
	cfg.SetProviderModelConfig("openai_subscription", config.ProviderModelConfig{DefaultModel: "gpt-5.5"})

	var attributedProvider string
	var attributedModel string
	var attributedUsage api.Usage
	first, err := SearchWeb(context.Background(), WebSearchRequest{
		Config:       cfg,
		MainProvider: "deepseek",
		Query:        query,
		UsageAttribution: func(provider, model string, usage api.Usage) {
			attributedProvider = provider
			attributedModel = model
			attributedUsage = usage
		},
	})
	if err != nil {
		t.Fatalf("first SearchWeb() error = %v", err)
	}
	if first.Provider != "openai_subscription" || first.Model != "gpt-5.5" {
		t.Fatalf("first provider/model = %s/%s, want openai_subscription/gpt-5.5", first.Provider, first.Model)
	}
	if attributedProvider != "openai_subscription" || attributedModel != "gpt-5.5" {
		t.Fatalf("usage owner = %s/%s, want openai_subscription/gpt-5.5", attributedProvider, attributedModel)
	}
	if attributedUsage.InputTokens != 31 || attributedUsage.OutputTokens != 9 {
		t.Fatalf("usage = %+v, want subscription token usage", attributedUsage)
	}

	cfg.WebSearch.Provider = "openai_subscription"
	second, err := SearchWeb(context.Background(), WebSearchRequest{
		Config:       cfg,
		MainProvider: "deepseek",
		Query:        query,
	})
	if err != nil {
		t.Fatalf("second SearchWeb() error = %v", err)
	}
	if !second.Cached {
		t.Fatal("second.Cached = false, want canonical cache hit")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want alias/canonical cache owner to be shared", calls)
	}
}

func TestSearchWeb_KimiK27FallsBackToK26ForUsageAttribution(t *testing.T) {
	resetWebSearchCacheForTest()
	query := "noninteractive kimi k2.7 fallback"
	websearch.RegisterWithContextForTest(t, "kimi", func(ctx context.Context, gotQuery, model string) (string, error) {
		if gotQuery != query {
			t.Fatalf("query = %q, want %q", gotQuery, query)
		}
		if model != "kimi-k2.6" {
			t.Fatalf("model = %q, want kimi-k2.6 fallback", model)
		}
		callback := websearch.UsageCallbackFromContext(ctx)
		if callback == nil {
			t.Fatal("UsageCallbackFromContext() = nil, want callback")
		}
		callback(api.Usage{InputTokens: 23, OutputTokens: 8, CachedInputTokens: 5})
		return "1. Kimi\n   URL: https://platform.kimi.ai/docs/guide/use-web-search", nil
	})

	var attributedProvider string
	var attributedModel string
	var attributedUsage api.Usage
	got, err := SearchWeb(context.Background(), WebSearchRequest{
		Config:       config.DefaultConfig(),
		MainProvider: "kimi",
		MainModel:    "kimi-k2.7-code",
		Query:        query,
		UsageAttribution: func(provider, model string, usage api.Usage) {
			attributedProvider = provider
			attributedModel = model
			attributedUsage = usage
		},
	})
	if err != nil {
		t.Fatalf("SearchWeb() error = %v", err)
	}
	if got.Model != "kimi-k2.6" {
		t.Fatalf("SearchWeb().Model = %q, want kimi-k2.6 fallback", got.Model)
	}
	if attributedProvider != "kimi" || attributedModel != "kimi-k2.6" {
		t.Fatalf("usage owner = %s/%s, want kimi/kimi-k2.6", attributedProvider, attributedModel)
	}
	if attributedUsage.InputTokens != 23 || attributedUsage.OutputTokens != 8 || attributedUsage.CachedInputTokens != 5 {
		t.Fatalf("attributed usage = %+v, want fallback model usage", attributedUsage)
	}
}

func TestSearchWeb_KimiK27ConfigDefaultFallsBackToK26ForUsageAttribution(t *testing.T) {
	resetWebSearchCacheForTest()
	query := "noninteractive kimi config default k2.7 fallback"
	websearch.RegisterWithContextForTest(t, "kimi", func(ctx context.Context, gotQuery, model string) (string, error) {
		if gotQuery != query {
			t.Fatalf("query = %q, want %q", gotQuery, query)
		}
		if model != "kimi-k2.6" {
			t.Fatalf("model = %q, want kimi-k2.6 fallback from configured K2.7 default", model)
		}
		callback := websearch.UsageCallbackFromContext(ctx)
		if callback == nil {
			t.Fatal("UsageCallbackFromContext() = nil, want callback")
		}
		callback(api.Usage{InputTokens: 29, OutputTokens: 11, CachedInputTokens: 7})
		return "1. Kimi\n   URL: https://platform.kimi.ai/docs/guide/use-web-search", nil
	})

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("kimi", config.ProviderModelConfig{DefaultModel: "kimi-k2.7-code"})

	var attributedProvider string
	var attributedModel string
	var attributedUsage api.Usage
	got, err := SearchWeb(context.Background(), WebSearchRequest{
		Config:       cfg,
		MainProvider: "kimi",
		Query:        query,
		UsageAttribution: func(provider, model string, usage api.Usage) {
			attributedProvider = provider
			attributedModel = model
			attributedUsage = usage
		},
	})
	if err != nil {
		t.Fatalf("SearchWeb() error = %v", err)
	}
	if got.Model != "kimi-k2.6" {
		t.Fatalf("SearchWeb().Model = %q, want kimi-k2.6 fallback", got.Model)
	}
	if attributedProvider != "kimi" || attributedModel != "kimi-k2.6" {
		t.Fatalf("usage owner = %s/%s, want kimi/kimi-k2.6", attributedProvider, attributedModel)
	}
	if attributedUsage.InputTokens != 29 || attributedUsage.OutputTokens != 11 || attributedUsage.CachedInputTokens != 7 {
		t.Fatalf("attributed usage = %+v, want config-driven fallback model usage", attributedUsage)
	}
}

func TestSearchWebUsesCache(t *testing.T) {
	resetWebSearchCacheForTest()
	provider := "gemini"
	calls := 0
	websearch.RegisterWithContextForTest(t, provider, func(_ context.Context, query, _ string) (string, error) {
		calls++
		return fmt.Sprintf("1. Docs\n   URL: https://docs.example.test/%d?q=%s", calls, query), nil
	})
	cfg := config.DefaultConfig()
	cfg.WebSearch.Provider = provider

	first, err := SearchWeb(context.Background(), WebSearchRequest{Config: cfg, Query: "cache query"})
	if err != nil {
		t.Fatalf("first SearchWeb() error = %v", err)
	}
	second, err := SearchWeb(context.Background(), WebSearchRequest{Config: cfg, Query: "cache query"})
	if err != nil {
		t.Fatalf("second SearchWeb() error = %v", err)
	}
	if first.Cached {
		t.Fatal("first.Cached = true, want false")
	}
	if !second.Cached {
		t.Fatal("second.Cached = false, want true")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want cached single provider call", calls)
	}
}
