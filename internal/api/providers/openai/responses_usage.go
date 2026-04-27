package openai

import "github.com/susugadx/xelyon-cli/internal/api"

func responsesUsageToAPIUsage(usage *ResponsesUsage) *api.Usage {
	if usage == nil {
		return nil
	}

	cachedTokens := 0
	if usage.InputTokensDetails != nil {
		cachedTokens = usage.InputTokensDetails.CachedTokens
	}
	reasoningTokens := 0
	if usage.OutputTokensDetails != nil {
		reasoningTokens = usage.OutputTokensDetails.ReasoningTokens
	}
	apiUsage := api.UsageFromOutputTokensIncludingThinking(usage.InputTokens, usage.OutputTokens, cachedTokens, reasoningTokens)
	return &apiUsage
}
