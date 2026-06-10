package cost

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

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
	"openai_subscription": func(pricingRequest) PricingInfo {
		return pricingUnavailableInfo()
	},
	"claude": func(req pricingRequest) PricingInfo {
		return getClaudePricing(req.Model, req.PromptTokenCount)
	},
	"bedrock": func(req pricingRequest) PricingInfo {
		return getBedrockPricing(req.Model, req.PromptTokenCount)
	},
	geminiPricingFamilyStandard: func(req pricingRequest) PricingInfo {
		return getGeminiStandardPricing(req.Model, req.PromptTokenCount)
	},
	geminiPricingFamilyFlex: func(req pricingRequest) PricingInfo {
		return getGeminiPricing(req.Model, req.PromptTokenCount, config.GeminiServiceTierFlex)
	},
	geminiPricingFamilyPriority: func(req pricingRequest) PricingInfo {
		return getGeminiPricing(req.Model, req.PromptTokenCount, config.GeminiServiceTierPriority)
	},
	"groq": func(req pricingRequest) PricingInfo {
		return getGroqPricing(req.Model)
	},
	"kimi": func(req pricingRequest) PricingInfo {
		return getKimiPricing(req.Model)
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

// GetPricingInfoForConfig は catalog_model 設定を考慮して料金情報を返す。
func GetPricingInfoForConfig(cfg *config.Config, provider string, model string, promptTokenCount ...int) PricingInfo {
	resolution := pricingModelResolutionForConfig(cfg, provider, model)
	if resolution.ConfiguredWithoutCatalog && !configuredModelCanUseDirectPricing(provider, resolution.Model) {
		return pricingUnavailableInfo()
	}
	return resolvePricingByProviderForConfig(cfg, provider, resolution.Model, normalizePromptTokenCount(promptTokenCount))
}

// HasKnownPricingModel は provider の pricing family が model を exact known model として持つか返す。
func HasKnownPricingModel(provider string, model string) bool {
	if strings.TrimSpace(model) == "" {
		return false
	}
	route := llmcatalog.ResolveProviderRoute(provider, model, model)
	if route.PricingFamily == "" {
		return false
	}
	return pricingFamilyHasKnownModel(route.PricingFamily, model)
}

func pricingModelForConfig(cfg *config.Config, provider string, model string) string {
	return pricingModelResolutionForConfig(cfg, provider, model).Model
}

func pricingModelResolutionForConfig(cfg *config.Config, provider string, model string) config.ModelCatalogResolution {
	if cfg == nil {
		return config.ModelCatalogResolution{Model: model}
	}
	return cfg.ResolveModelCatalog(provider, model)
}

func configuredModelCanUseDirectPricing(provider, model string) bool {
	route := llmcatalog.ResolveProviderRoute(provider, model, model)
	if route.PricingFamily == "" {
		return false
	}
	if route.PricingFamily == "ollama" {
		return true
	}
	return pricingFamilyHasKnownModel(route.PricingFamily, model)
}

func normalizePromptTokenCount(promptTokenCount []int) int {
	if len(promptTokenCount) == 0 {
		return 0
	}
	return promptTokenCount[0]
}

func resolvePricingByProvider(provider string, model string, promptTokenCount int) PricingInfo {
	route := llmcatalog.ResolveProviderRoute(provider, model, model)
	if route.PricingFamily == "" {
		return pricingUnavailableInfo()
	}

	return resolvePricingByFamily(route.PricingFamily, pricingRequest{
		Model:            model,
		PromptTokenCount: promptTokenCount,
	})
}

func resolvePricingByProviderForConfig(cfg *config.Config, provider string, model string, promptTokenCount int) PricingInfo {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	route := llmcatalog.ResolveProviderRoute(provider, model, model)
	if route.PricingFamily == "" {
		return pricingUnavailableInfo()
	}
	family := route.PricingFamily
	if family == geminiPricingFamilyStandard {
		family = geminiPricingFamilyForServiceTier(cfg.GeminiServiceTier())
	}
	return resolvePricingByFamily(family, pricingRequest{
		Model:            model,
		PromptTokenCount: promptTokenCount,
	})
}

func resolvePricingByFamily(family string, req pricingRequest) PricingInfo {
	resolver, ok := pricingResolvers[family]
	if !ok {
		if pricing, loaded := resolveProviderPricingFromLoadedConfig(family, req.Model, req.PromptTokenCount, true); loaded {
			return pricing
		}
		return pricingUnavailableInfo()
	}
	return resolver(req)
}
