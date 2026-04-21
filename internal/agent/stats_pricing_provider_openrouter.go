package agent

import (
	"strings"
)

func getOpenRouterPricing(model string, promptTokenCount int) PricingInfo {
	// OpenRouterのモデル名形式: "anthropic/claude-opus-4.6", "google/gemini-3.1-pro" 等
	lm := strings.ToLower(model)
	switch {
	case strings.Contains(lm, "claude"):
		return getClaudePricing(model, promptTokenCount)
	case strings.Contains(lm, "gpt") || strings.Contains(lm, "openai") || strings.Contains(lm, "codex"):
		return getOpenAIPricing(model, promptTokenCount)
	case strings.Contains(lm, "gemini") || strings.Contains(lm, "google"):
		return getGeminiPricing(model, promptTokenCount)
	case strings.Contains(lm, "deepseek"):
		return getDeepSeekPricing(model)
	case strings.Contains(lm, "kimi") || strings.Contains(lm, "moonshotai"):
		return getKimiPricing(model)
	case strings.Contains(lm, "mistral") || strings.Contains(lm, "codestral"):
		return PricingInfo{InputCostPerM: 2.00, OutputCostPerM: 6.00, CachedInputCostPerM: 0.20, CacheCreationCostPerM: 2.00}
	case strings.Contains(lm, "llama") || strings.Contains(lm, "meta"):
		return PricingInfo{InputCostPerM: 0.20, OutputCostPerM: 0.80, CachedInputCostPerM: 0.02, CacheCreationCostPerM: 0.20}
	case strings.Contains(lm, "qwen"):
		return PricingInfo{InputCostPerM: 0.15, OutputCostPerM: 0.60, CachedInputCostPerM: 0.015, CacheCreationCostPerM: 0.15}
	case strings.Contains(lm, "glm-5"):
		return PricingInfo{InputCostPerM: 0.72, OutputCostPerM: 2.30, CachedInputCostPerM: 0.072, CacheCreationCostPerM: 0.72}
	default:
		return getDeepSeekPricing(model)
	}
}
