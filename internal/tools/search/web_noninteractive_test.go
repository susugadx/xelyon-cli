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
