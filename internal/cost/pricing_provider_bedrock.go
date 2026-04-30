package cost

import (
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

func getBedrockPricing(model string, promptTokenCount int) PricingInfo {
	if normalizePricingModelName(model) == "" {
		return pricingUnavailableInfo()
	}

	if pricingFamilyHasKnownModel("bedrock", model) {
		if pricing, ok := resolveProviderPricingFromLoadedConfig("bedrock", model, promptTokenCount, false); ok {
			return pricing
		}
		return pricingUnavailableInfo()
	}

	if bedrockModelCanUseClaudePricing(model) {
		return getClaudePricing(model, promptTokenCount)
	}

	if pricing, ok := resolveProviderPricingFromLoadedConfig("bedrock", model, promptTokenCount, false); ok {
		return pricing
	}

	return pricingUnavailableInfo()
}

func bedrockModelCanUseClaudePricing(model string) bool {
	if !llmcatalog.IsBedrockClaudeModel(model) {
		return false
	}
	return pricingFamilyHasKnownModel("bedrock", model) ||
		pricingFamilyHasKnownModel("claude", model)
}
