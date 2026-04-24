package cost

import "strings"

func resolveOpenRouterDelegatedProviderPricing(model string, promptTokenCount int) (PricingInfo, bool) {
	// OpenRouterのモデル名形式: "anthropic/claude-opus-4.6", "google/gemini-3.1-pro" 等
	lm := strings.ToLower(model)
	switch {
	case strings.Contains(lm, "claude"):
		return getClaudePricing(model, promptTokenCount), true
	case strings.Contains(lm, "gpt") || strings.Contains(lm, "openai") || strings.Contains(lm, "codex"):
		return getOpenAIPricing(model, promptTokenCount), true
	case strings.Contains(lm, "gemini") || strings.Contains(lm, "google"):
		return getGeminiPricing(model, promptTokenCount), true
	case strings.Contains(lm, "deepseek"):
		return getDeepSeekPricing(model), true
	case strings.Contains(lm, "kimi") || strings.Contains(lm, "moonshotai"):
		return getKimiPricing(model), true
	default:
		return PricingInfo{}, false
	}
}
