package cost

import (
	"strings"
)

// getKimiPricing はモデル名からKimi料金を返す
func getKimiPricing(model string) PricingInfo {
	lm := strings.ToLower(model)
	if pricing, ok := resolveProviderPricingFromLoadedConfig("kimi", lm, 0, false); ok {
		return pricing
	}

	if strings.Contains(lm, "k2.5") {
		// Kimi K2.5: $0.60/$3.00 per million tokens
		return PricingInfo{
			InputCostPerM:         0.60,
			OutputCostPerM:        3.00,
			CachedInputCostPerM:   0.06,
			CacheCreationCostPerM: 0.60,
		}
	}
	if lm == "" || strings.Contains(lm, "k2") {
		// Kimi K2（デフォルト）: $0.60/$2.50 per million tokens
		return PricingInfo{
			InputCostPerM:         0.60,
			OutputCostPerM:        2.50,
			CachedInputCostPerM:   0.06,
			CacheCreationCostPerM: 0.60,
		}
	}
	return pricingUnavailableInfo()
}
