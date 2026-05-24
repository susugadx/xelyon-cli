package cost

import (
	"strings"
)

// getGeminiPricing はモデル名からGemini料金を返す
// promptTokenCount はリクエストの入力トークン数（200Kティア判定に使用）
func getGeminiPricing(model string, promptTokenCount int) PricingInfo {
	lm := strings.ToLower(model)
	if pricing, ok := resolveProviderPricingFromLoadedConfig("gemini", lm, promptTokenCount, true); ok {
		return pricing
	}

	switch {
	case strings.Contains(lm, "2.5-pro"):
		if promptTokenCount > 200000 {
			return PricingInfo{
				InputCostPerM: 2.50, OutputCostPerM: 15.00,
				CachedInputCostPerM: 0.25, CacheCreationCostPerM: 2.50,
			}
		}
		return PricingInfo{
			InputCostPerM: 1.25, OutputCostPerM: 10.00,
			CachedInputCostPerM: 0.125, CacheCreationCostPerM: 1.25,
		}
	case strings.Contains(lm, "pro"):
		if promptTokenCount > 200000 {
			// Long context pricing (>200K): $4/$18 per million tokens
			return PricingInfo{
				InputCostPerM: 4.00, OutputCostPerM: 18.00,
				CachedInputCostPerM: 0.40, CacheCreationCostPerM: 4.00,
			}
		}
		// Gemini 3.x Pro (<=200K): $2/$12 per million tokens
		return PricingInfo{
			InputCostPerM: 2.00, OutputCostPerM: 12.00,
			CachedInputCostPerM: 0.20, CacheCreationCostPerM: 2.00,
		}
	case strings.Contains(lm, "3.5-flash"):
		return PricingInfo{
			InputCostPerM: 1.50, OutputCostPerM: 9.00,
			CachedInputCostPerM: 0.15, CacheCreationCostPerM: 1.50,
		}
	case strings.Contains(lm, "3.1-flash-lite"):
		// Gemini 3.1 Flash-Lite: $0.25/$1.50 per million tokens
		return PricingInfo{
			InputCostPerM: 0.25, OutputCostPerM: 1.50,
			CachedInputCostPerM: 0.025, CacheCreationCostPerM: 0.25,
		}
	case strings.Contains(lm, "2.5-flash-lite"):
		return PricingInfo{
			InputCostPerM: 0.10, OutputCostPerM: 0.40,
			CachedInputCostPerM: 0.01, CacheCreationCostPerM: 0.10,
		}
	case strings.Contains(lm, "2.0-flash-lite"):
		return PricingInfo{
			InputCostPerM: 0.075, OutputCostPerM: 0.30,
			CachedInputCostPerM: 0.01, CacheCreationCostPerM: 0.075,
		}
	case strings.Contains(lm, "2.5-flash"):
		return PricingInfo{
			InputCostPerM: 0.30, OutputCostPerM: 2.50,
			CachedInputCostPerM: 0.03, CacheCreationCostPerM: 0.30,
		}
	case strings.Contains(lm, "2.0-flash"):
		return PricingInfo{
			InputCostPerM: 0.10, OutputCostPerM: 0.40,
			CachedInputCostPerM: 0.025, CacheCreationCostPerM: 0.10,
		}
	case lm == "" || strings.Contains(lm, "3.1-flash") || strings.Contains(lm, "3-flash"):
		// Gemini 3 Flash（デフォルト）: $0.50/$3.00 per million tokens
		return PricingInfo{
			InputCostPerM: 0.50, OutputCostPerM: 3.00,
			CachedInputCostPerM: 0.05, CacheCreationCostPerM: 0.50,
		}
	default:
		return pricingUnavailableInfo()
	}
}
