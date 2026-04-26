package cost

import "strings"

type staticFallbackPricingRule struct {
	contains []string
	pricing  PricingInfo
}

var openRouterStaticFallbackPricingRules = []staticFallbackPricingRule{
	{
		contains: []string{"mistral", "codestral"},
		pricing: PricingInfo{
			InputCostPerM:         2.00,
			OutputCostPerM:        6.00,
			CachedInputCostPerM:   0.20,
			CacheCreationCostPerM: 2.00,
		},
	},
	{
		contains: []string{"llama", "meta"},
		pricing: PricingInfo{
			InputCostPerM:         0.20,
			OutputCostPerM:        0.80,
			CachedInputCostPerM:   0.02,
			CacheCreationCostPerM: 0.20,
		},
	},
	{
		contains: []string{"qwen"},
		pricing: PricingInfo{
			InputCostPerM:         0.15,
			OutputCostPerM:        0.60,
			CachedInputCostPerM:   0.015,
			CacheCreationCostPerM: 0.15,
		},
	},
	{
		contains: []string{"glm-5"},
		pricing: PricingInfo{
			InputCostPerM:         0.72,
			OutputCostPerM:        2.30,
			CachedInputCostPerM:   0.072,
			CacheCreationCostPerM: 0.72,
		},
	},
}

func resolveOpenRouterStaticFallbackPricing(model string) (PricingInfo, bool) {
	lm := strings.ToLower(model)
	for _, rule := range openRouterStaticFallbackPricingRules {
		if containsAny(lm, rule.contains) {
			return rule.pricing, true
		}
	}
	return PricingInfo{}, false
}
