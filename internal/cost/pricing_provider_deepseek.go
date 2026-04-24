package cost

import (
	"strings"
)

// getDeepSeekPricing はモデル名からDeepSeek料金を返す
func getDeepSeekPricing(model string) PricingInfo {
	lm := strings.ToLower(model)
	if cfg := loadPricingConfig(); cfg != nil {
		return resolveProviderPricingFromConfig(cfg.DeepSeek, lm, 0, false)
	}

	// DeepSeek V3.2: deepseek-chat/deepseek-reasoner 統一料金
	// $0.28/$0.42 per million tokens, Cache hit: $0.028
	return PricingInfo{
		InputCostPerM:         0.28,
		OutputCostPerM:        0.42,
		CachedInputCostPerM:   0.028,
		CacheCreationCostPerM: 0.28,
	}
}
