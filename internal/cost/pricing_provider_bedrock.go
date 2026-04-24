package cost

import (
	"strings"
)

func getBedrockPricing(model string, promptTokenCount int) PricingInfo {
	if strings.Contains(strings.ToLower(model), "claude") {
		return getClaudePricing(model, promptTokenCount)
	}

	// Claude以外のBedrockモデルは一旦汎用料金
	return getUnknownProviderFallbackPricing()
}
