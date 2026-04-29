package cost

type openRouterDelegationRule struct {
	owners   []string
	resolver pricingResolver
}

var openRouterDelegationRules = []openRouterDelegationRule{
	{
		owners: []string{"anthropic"},
		resolver: func(req pricingRequest) PricingInfo {
			return getClaudePricing(req.Model, req.PromptTokenCount)
		},
	},
	{
		owners: []string{"openai"},
		resolver: func(req pricingRequest) PricingInfo {
			return getOpenAIPricing(req.Model, req.PromptTokenCount)
		},
	},
	{
		owners: []string{"google"},
		resolver: func(req pricingRequest) PricingInfo {
			return getGeminiPricing(req.Model, req.PromptTokenCount)
		},
	},
	{
		owners: []string{"deepseek"},
		resolver: func(req pricingRequest) PricingInfo {
			return getDeepSeekPricing(req.Model)
		},
	},
	{
		owners: []string{"moonshotai"},
		resolver: func(req pricingRequest) PricingInfo {
			return getKimiPricing(req.Model)
		},
	},
}

func resolveOpenRouterDelegatedProviderPricing(model string, promptTokenCount int) (PricingInfo, bool) {
	// OpenRouterのモデル名形式: "anthropic/claude-opus-4.6", "google/gemini-3.1-pro" 等
	id, ok := parseOpenRouterModelID(model)
	if !ok || !pricingFamilyHasKnownModel("openrouter", model) {
		return PricingInfo{}, false
	}
	for _, rule := range openRouterDelegationRules {
		if !rule.matchesOwner(id.owner) {
			continue
		}
		return rule.resolver(pricingRequest{
			Model:            id.routedModel,
			PromptTokenCount: promptTokenCount,
		}), true
	}
	return PricingInfo{}, false
}

func (rule openRouterDelegationRule) matchesOwner(owner string) bool {
	for _, candidate := range rule.owners {
		if owner == candidate {
			return true
		}
	}
	return false
}
