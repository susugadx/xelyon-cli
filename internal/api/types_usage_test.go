package api

import "testing"

func TestUsageFromOutputTokensIncludingThinking(t *testing.T) {
	usage := UsageFromOutputTokensIncludingThinking(100, 50, 10, 20)

	if usage.InputTokens != 100 || usage.OutputTokens != 30 || usage.CachedInputTokens != 10 || usage.ThinkingTokens != 20 {
		t.Fatalf("UsageFromOutputTokensIncludingThinking() = %+v, want input=100 output=30 cached=10 thinking=20", usage)
	}
}

func TestUsageFromOutputTokensIncludingThinking_ClampsOutputAtZero(t *testing.T) {
	usage := UsageFromOutputTokensIncludingThinking(100, 10, 0, 20)

	if usage.OutputTokens != 0 || usage.ThinkingTokens != 20 {
		t.Fatalf("UsageFromOutputTokensIncludingThinking() = %+v, want output=0 thinking=20", usage)
	}
}

func TestUsageAdd(t *testing.T) {
	usage := Usage{
		InputTokens:           10,
		OutputTokens:          20,
		ThinkingTokens:        5,
		CachedInputTokens:     2,
		CacheCreationTokens:   1,
		StorageCost:           0.01,
		BillingServiceTier:    "priority",
		WebSearchCalls:        1,
		WebSearchResultTokens: 100,
	}

	usage.Add(Usage{
		InputTokens:           3,
		OutputTokens:          4,
		ThinkingTokens:        6,
		CachedInputTokens:     7,
		CacheCreationTokens:   8,
		StorageCost:           0.02,
		BillingServiceTier:    "standard",
		WebSearchCalls:        2,
		WebSearchResultTokens: 300,
	})

	if usage.InputTokens != 13 ||
		usage.OutputTokens != 24 ||
		usage.ThinkingTokens != 11 ||
		usage.CachedInputTokens != 9 ||
		usage.CacheCreationTokens != 9 ||
		usage.StorageCost < 0.0299 ||
		usage.StorageCost > 0.0301 ||
		usage.BillingServiceTier != "standard" ||
		usage.WebSearchCalls != 3 ||
		usage.WebSearchResultTokens != 400 {
		t.Fatalf("Usage.Add() = %+v, want all fields accumulated", usage)
	}
}

func TestUsageHasTokenOrWebSearchObservation(t *testing.T) {
	if (Usage{}).HasTokenOrWebSearchObservation() {
		t.Fatal("empty usage should not have token or web search observation")
	}
	if (Usage{StorageCost: 0.005}).HasTokenOrWebSearchObservation() {
		t.Fatal("storage cost without explicit web search call should not be treated as token or web search observation")
	}
	if !(Usage{InputTokens: 1}).HasTokenOrWebSearchObservation() {
		t.Fatal("token usage should be observed")
	}
	if !(Usage{WebSearchCalls: 1}).HasTokenOrWebSearchObservation() {
		t.Fatal("web search call usage should be observed")
	}
	if !(Usage{WebSearchResultTokens: 1}).HasTokenOrWebSearchObservation() {
		t.Fatal("web search result token observation should be observed")
	}
}

func TestUsageHasTokenObservation(t *testing.T) {
	if (Usage{WebSearchCalls: 1, StorageCost: 0.005}).HasTokenObservation() {
		t.Fatal("synthetic web search fee should not be treated as endpoint token usage")
	}
	for _, usage := range []Usage{
		{InputTokens: 1},
		{OutputTokens: 1},
		{ThinkingTokens: 1},
		{CachedInputTokens: 1},
		{CacheCreationTokens: 1},
	} {
		if !usage.HasTokenObservation() {
			t.Fatalf("HasTokenObservation() = false for %+v, want true", usage)
		}
	}
}

func TestStreamUsageInfoToUsage(t *testing.T) {
	raw := StreamUsageInfo{
		PromptTokens:     100,
		CompletionTokens: 50,
		PromptTokensDetails: &PromptTokensDetails{
			CachedTokens: 10,
		},
		CompletionTokensDetails: &CompletionTokensDetails{
			ReasoningTokens: 20,
		},
	}

	usage := raw.ToUsage()
	if usage.InputTokens != 100 || usage.OutputTokens != 30 || usage.CachedInputTokens != 10 || usage.ThinkingTokens != 20 {
		t.Fatalf("ToUsage() = %+v, want input=100 output=30 cached=10 thinking=20", usage)
	}
}
