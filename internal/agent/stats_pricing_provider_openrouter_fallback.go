package agent

import "strings"

func resolveOpenRouterStaticFallbackPricing(model string) (PricingInfo, bool) {
	lm := strings.ToLower(model)
	switch {
	case strings.Contains(lm, "mistral") || strings.Contains(lm, "codestral"):
		return PricingInfo{
			InputCostPerM:         2.00,
			OutputCostPerM:        6.00,
			CachedInputCostPerM:   0.20,
			CacheCreationCostPerM: 2.00,
		}, true
	case strings.Contains(lm, "llama") || strings.Contains(lm, "meta"):
		return PricingInfo{
			InputCostPerM:         0.20,
			OutputCostPerM:        0.80,
			CachedInputCostPerM:   0.02,
			CacheCreationCostPerM: 0.20,
		}, true
	case strings.Contains(lm, "qwen"):
		return PricingInfo{
			InputCostPerM:         0.15,
			OutputCostPerM:        0.60,
			CachedInputCostPerM:   0.015,
			CacheCreationCostPerM: 0.15,
		}, true
	case strings.Contains(lm, "glm-5"):
		return PricingInfo{
			InputCostPerM:         0.72,
			OutputCostPerM:        2.30,
			CachedInputCostPerM:   0.072,
			CacheCreationCostPerM: 0.72,
		}, true
	default:
		return PricingInfo{}, false
	}
}
