package cost

import "strings"

type openRouterDelegationRule struct {
	contains []string
	resolver pricingResolver
}

var openRouterDelegationRules = []openRouterDelegationRule{
	{
		contains: []string{"claude"},
		resolver: func(req pricingRequest) PricingInfo {
			return getClaudePricing(req.Model, req.PromptTokenCount)
		},
	},
	{
		contains: []string{"gpt", "openai", "codex"},
		resolver: func(req pricingRequest) PricingInfo {
			return getOpenAIPricing(req.Model, req.PromptTokenCount)
		},
	},
	{
		contains: []string{"gemini", "google"},
		resolver: func(req pricingRequest) PricingInfo {
			return getGeminiPricing(req.Model, req.PromptTokenCount)
		},
	},
	{
		contains: []string{"deepseek"},
		resolver: func(req pricingRequest) PricingInfo {
			return getDeepSeekPricing(req.Model)
		},
	},
	{
		contains: []string{"kimi", "moonshotai"},
		resolver: func(req pricingRequest) PricingInfo {
			return getKimiPricing(req.Model)
		},
	},
}

func resolveOpenRouterDelegatedProviderPricing(model string, promptTokenCount int) (PricingInfo, bool) {
	// OpenRouterのモデル名形式: "anthropic/claude-opus-4.6", "google/gemini-3.1-pro" 等
	lm := strings.ToLower(model)
	for _, rule := range openRouterDelegationRules {
		if !containsAny(lm, rule.contains) {
			continue
		}
		return rule.resolver(pricingRequest{
			Model:            model,
			PromptTokenCount: promptTokenCount,
		}), true
	}
	return PricingInfo{}, false
}

func containsAny(value string, candidates []string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}
