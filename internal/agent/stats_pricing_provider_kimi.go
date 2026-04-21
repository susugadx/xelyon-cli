package agent

import (
	"strings"
)

// getKimiPricing はモデル名からKimi料金を返す
func getKimiPricing(model string) PricingInfo {
	lm := strings.ToLower(model)
	if cfg := loadPricingConfig(); cfg != nil {
		return resolveProviderPricingFromConfig(cfg.Kimi, lm, 0, false)
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
	// Kimi K2（デフォルト）: $0.60/$2.50 per million tokens
	return PricingInfo{
		InputCostPerM:         0.60,
		OutputCostPerM:        2.50,
		CachedInputCostPerM:   0.06,
		CacheCreationCostPerM: 0.60,
	}
}
