package cost

import "github.com/susugadx/xelyon-cli/internal/config"

// GetPricingInfo はプロバイダー・モデル別の料金情報を返す
// promptTokenCount はオプション（Gemini 200Kティア判定用）
func GetPricingInfo(provider string, model string, promptTokenCount ...int) PricingInfo {
	return resolvePricingByCanonicalProvider(config.CanonicalProviderName(provider), model, normalizePromptTokenCount(promptTokenCount))
}

func normalizePromptTokenCount(promptTokenCount []int) int {
	if len(promptTokenCount) == 0 {
		return 0
	}
	return promptTokenCount[0]
}

func resolvePricingByCanonicalProvider(provider string, model string, promptTokenCount int) PricingInfo {
	switch provider {
	case "deepseek":
		// DeepSeekの料金体系
		return getDeepSeekPricing(model)
	case "openai":
		return getOpenAIPricing(model, promptTokenCount)
	case "claude":
		return getClaudePricing(model, promptTokenCount)
	case "bedrock":
		return getBedrockPricing(model, promptTokenCount)
	case "gemini":
		return getGeminiPricing(model, promptTokenCount)
	case "groq":
		return getGroqPricing(model)
	case "openrouter":
		return getOpenRouterPricing(model, promptTokenCount)
	case "ollama":
		return PricingInfo{} // ローカル実行は無料
	default:
		return getUnknownProviderFallbackPricing()
	}
}
