package cost

import "github.com/susugadx/xelyon-cli/internal/llmcatalog"

type pricingRequest struct {
	Model            string
	PromptTokenCount int
}

type pricingResolver func(pricingRequest) PricingInfo

var pricingResolvers = map[string]pricingResolver{
	"deepseek": func(req pricingRequest) PricingInfo {
		return getDeepSeekPricing(req.Model)
	},
	"openai": func(req pricingRequest) PricingInfo {
		return getOpenAIPricing(req.Model, req.PromptTokenCount)
	},
	"claude": func(req pricingRequest) PricingInfo {
		return getClaudePricing(req.Model, req.PromptTokenCount)
	},
	"bedrock": func(req pricingRequest) PricingInfo {
		return getBedrockPricing(req.Model, req.PromptTokenCount)
	},
	"gemini": func(req pricingRequest) PricingInfo {
		return getGeminiPricing(req.Model, req.PromptTokenCount)
	},
	"groq": func(req pricingRequest) PricingInfo {
		return getGroqPricing(req.Model)
	},
	"openrouter": func(req pricingRequest) PricingInfo {
		return getOpenRouterPricing(req.Model, req.PromptTokenCount)
	},
	"ollama": func(pricingRequest) PricingInfo {
		return PricingInfo{}
	},
}

// GetPricingInfo はプロバイダー・モデル別の料金情報を返す
// promptTokenCount はオプション（Gemini 200Kティア判定用）
func GetPricingInfo(provider string, model string, promptTokenCount ...int) PricingInfo {
	return resolvePricingByProvider(provider, model, normalizePromptTokenCount(promptTokenCount))
}

func normalizePromptTokenCount(promptTokenCount []int) int {
	if len(promptTokenCount) == 0 {
		return 0
	}
	return promptTokenCount[0]
}

func resolvePricingByProvider(provider string, model string, promptTokenCount int) PricingInfo {
	entry, ok := llmcatalog.ProviderDescriptorFor(provider)
	if !ok {
		return getUnknownProviderFallbackPricing()
	}

	return resolvePricingByFamily(entry.PricingFamily, pricingRequest{
		Model:            model,
		PromptTokenCount: promptTokenCount,
	})
}

func resolvePricingByFamily(family string, req pricingRequest) PricingInfo {
	resolver, ok := pricingResolvers[family]
	if !ok {
		return getUnknownProviderFallbackPricing()
	}
	return resolver(req)
}
