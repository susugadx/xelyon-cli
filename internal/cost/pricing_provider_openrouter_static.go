package cost

import "strings"

type openRouterStaticPricingRule struct {
	modelIDs []string
	pricing  PricingInfo
}

var openRouterStaticPricingRules = []openRouterStaticPricingRule{
	{
		modelIDs: []string{"mistral-medium"},
		pricing: PricingInfo{
			InputCostPerM:         2.00,
			OutputCostPerM:        6.00,
			CachedInputCostPerM:   0.20,
			CacheCreationCostPerM: 2.00,
		},
	},
	{
		modelIDs: []string{"meta/llama-3.1-70b"},
		pricing: PricingInfo{
			InputCostPerM:         0.20,
			OutputCostPerM:        0.80,
			CachedInputCostPerM:   0.02,
			CacheCreationCostPerM: 0.20,
		},
	},
	{
		modelIDs: []string{"qwen/qwen2.5-coder"},
		pricing: PricingInfo{
			InputCostPerM:         0.15,
			OutputCostPerM:        0.60,
			CachedInputCostPerM:   0.015,
			CacheCreationCostPerM: 0.15,
		},
	},
	{
		modelIDs: []string{"zhipu/glm-5"},
		pricing: PricingInfo{
			InputCostPerM:         0.72,
			OutputCostPerM:        2.30,
			CachedInputCostPerM:   0.072,
			CacheCreationCostPerM: 0.72,
		},
	},
}

func resolveOpenRouterStaticPricing(model string) (PricingInfo, bool) {
	if !pricingFamilyHasKnownModel("openrouter", model) {
		return PricingInfo{}, false
	}
	lm := strings.ToLower(strings.TrimSpace(model))
	for _, rule := range openRouterStaticPricingRules {
		if rule.matchesModel(lm) {
			return rule.pricing, true
		}
	}
	return PricingInfo{}, false
}

func (rule openRouterStaticPricingRule) matchesModel(model string) bool {
	for _, candidate := range rule.modelIDs {
		if model == candidate {
			return true
		}
	}
	return false
}
