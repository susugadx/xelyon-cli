package gemini

import "testing"

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
