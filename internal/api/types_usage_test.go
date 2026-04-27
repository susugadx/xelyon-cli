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
