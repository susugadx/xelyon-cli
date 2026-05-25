package gemini

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestGeminiUsageMetadataToAPIUsageNormalizesTokenUsage(t *testing.T) {
	usage, ok := geminiUsageMetadataToAPIUsage(&GeminiUsageMetadata{
		PromptTokenCount:        17,
		CandidatesTokenCount:    5,
		ThoughtsTokenCount:      3,
		CachedContentTokenCount: 4,
	})
	if !ok {
		t.Fatal("geminiUsageMetadataToAPIUsage() ok = false, want true")
	}
	if usage.InputTokens != 17 ||
		usage.OutputTokens != 5 ||
		usage.ThinkingTokens != 3 ||
		usage.CachedInputTokens != 4 ||
		usage.WebSearchCalls != 0 ||
		usage.StorageCost != 0 ||
		usage.WebSearchResultTokens != 0 {
		t.Fatalf("usage = %+v, want Gemini usageMetadata as token usage without web search fee fields", usage)
	}
}

func TestGeminiUsageMetadataToAPIUsageNilIsUnobserved(t *testing.T) {
	usage, ok := geminiUsageMetadataToAPIUsage(nil)
	if ok {
		t.Fatalf("geminiUsageMetadataToAPIUsage(nil) ok = true, want false: %+v", usage)
	}
}

func TestGeminiUsageMetadataToAPIUsageUsesMetadataServiceTier(t *testing.T) {
	usage, ok := geminiUsageMetadataToAPIUsage(&GeminiUsageMetadata{
		PromptTokenCount:     10,
		CandidatesTokenCount: 5,
		ServiceTier:          config.GeminiServiceTierStandard,
	}, config.GeminiServiceTierPriority)
	if !ok {
		t.Fatal("geminiUsageMetadataToAPIUsage() ok = false, want true")
	}
	if usage.BillingServiceTier != config.GeminiServiceTierStandard {
		t.Fatalf("BillingServiceTier = %q, want usageMetadata serviceTier", usage.BillingServiceTier)
	}
}

func TestGeminiUsageMetadataToAPIUsageFallsBackToHeaderTier(t *testing.T) {
	usage, ok := geminiUsageMetadataToAPIUsage(&GeminiUsageMetadata{
		PromptTokenCount:     10,
		CandidatesTokenCount: 5,
		ServiceTier:          "turbo",
	}, config.GeminiServiceTierStandard)
	if !ok {
		t.Fatal("geminiUsageMetadataToAPIUsage() ok = false, want true")
	}
	if usage.BillingServiceTier != config.GeminiServiceTierStandard {
		t.Fatalf("BillingServiceTier = %q, want header fallback tier", usage.BillingServiceTier)
	}
}

func TestGeminiBillingServiceTierUnspecifiedMeansStandard(t *testing.T) {
	if got := geminiBillingServiceTierFromValue("unspecified"); got != config.GeminiServiceTierStandard {
		t.Fatalf("geminiBillingServiceTierFromValue(unspecified) = %q, want standard", got)
	}
}
