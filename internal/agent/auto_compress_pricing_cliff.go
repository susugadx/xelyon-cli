package agent

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
)

func averageOutputTokens(stats *SessionStats) int {
	if stats == nil || stats.OutputTokens <= 0 {
		return 0
	}
	assistantMessages := stats.AssistantMessages
	if assistantMessages < 1 {
		assistantMessages = 1
	}
	return stats.OutputTokens / assistantMessages
}

func shouldForceCompressForPricingCliff(provider, model string, currentTokens int, stats *SessionStats) (int, bool) {
	return shouldForceCompressForPricingCliffForConfig(nil, provider, model, currentTokens, stats)
}

func shouldForceCompressForPricingCliffForConfig(cfg *config.Config, provider, model string, currentTokens int, stats *SessionStats) (int, bool) {
	if currentTokens <= 0 {
		return currentTokens, false
	}

	projectedTokens := currentTokens + averageOutputTokens(stats)
	basePricing := cost.GetPricingInfoForConfig(cfg, provider, model, 0)
	currentPricing := cost.GetPricingInfoForConfig(cfg, provider, model, currentTokens)
	projectedPricing := cost.GetPricingInfoForConfig(cfg, provider, model, projectedTokens)
	if basePricing.PricingUnavailable || currentPricing.PricingUnavailable || projectedPricing.PricingUnavailable {
		return projectedTokens, false
	}
	if currentPricing.InputCostPerM > basePricing.InputCostPerM {
		return projectedTokens, true
	}
	if projectedPricing.InputCostPerM > currentPricing.InputCostPerM {
		return projectedTokens, true
	}

	return projectedTokens, false
}
