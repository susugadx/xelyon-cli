package cost

import (
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

func getBedrockPricing(model string, promptTokenCount int) PricingInfo {
	if bedrockModelCanUseClaudePricing(model) {
		return getClaudePricing(model, promptTokenCount)
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
