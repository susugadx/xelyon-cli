package openairesponses

import "github.com/susugadx/xelyon-cli/internal/api"

// UsageToAPIUsage は Responses API usage を XELYON 共通 usage に変換する。
func UsageToAPIUsage(usage *Usage) *api.Usage {
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

func responsesUsageToAPIUsage(usage *Usage) *api.Usage {
	return UsageToAPIUsage(usage)
}
