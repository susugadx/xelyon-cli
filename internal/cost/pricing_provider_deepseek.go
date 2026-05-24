package cost

import (
	"strings"
)

// getDeepSeekPricing はモデル名からDeepSeek料金を返す
func getDeepSeekPricing(model string) PricingInfo {
	lm := strings.ToLower(model)
	if pricing, ok := resolveProviderPricingFromLoadedConfig("deepseek", lm, 0, false); ok {
		return pricing
	}

	if strings.Contains(lm, "deepseek-v4-pro") {
		return PricingInfo{
			InputCostPerM:         0.435,
			OutputCostPerM:        0.87,
			CachedInputCostPerM:   0.003625,
			CacheCreationCostPerM: 0.435,
		}
	}
	if strings.Contains(lm, "deepseek-v4-flash") ||
		strings.Contains(lm, "deepseek-chat") ||
		strings.Contains(lm, "deepseek-reasoner") {
		return PricingInfo{
			InputCostPerM:         0.14,
			OutputCostPerM:        0.28,
			CachedInputCostPerM:   0.0028,
			CacheCreationCostPerM: 0.14,
		}
	}

	if strings.Contains(lm, "deepseek-v3") ||
		strings.Contains(lm, "deepseek-r1") ||
		strings.Contains(lm, "deepseek-coder") {
		return PricingInfo{
			InputCostPerM:         0.28,
			OutputCostPerM:        0.42,
			CachedInputCostPerM:   0.028,
			CacheCreationCostPerM: 0.28,
		}
	}
	return pricingUnavailableInfo()
}
