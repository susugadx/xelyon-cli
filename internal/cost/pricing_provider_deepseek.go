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
			InputCostPerM:         1.74,
			OutputCostPerM:        3.48,
			CachedInputCostPerM:   0.0145,
			CacheCreationCostPerM: 1.74,
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

	// DeepSeek V3.2/R1/coder など、V4 alias ではないモデルの既存フォールバック。
	// $0.28/$0.42 per million tokens, Cache hit: $0.028
	return PricingInfo{
		InputCostPerM:         0.28,
		OutputCostPerM:        0.42,
		CachedInputCostPerM:   0.028,
		CacheCreationCostPerM: 0.28,
	}
}
