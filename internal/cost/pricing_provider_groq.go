package cost

import (
	"strings"
)

// getGroqPricing はモデル名からGroq料金を返す
func getGroqPricing(model string) PricingInfo {
	lm := strings.ToLower(model)
	if pricing, ok := resolveProviderPricingFromLoadedConfig("groq", lm, 0, false); ok {
		return pricing
	}

	switch {
	case strings.Contains(lm, "70b"):
		// Llama 3/3.1 70B: $0.59/$0.79 per million tokens
		return PricingInfo{
			InputCostPerM:         0.59,
			OutputCostPerM:        0.79,
			CachedInputCostPerM:   0.59, // キャッシュ割引なし
			CacheCreationCostPerM: 0.59,
		}
	case strings.Contains(lm, "405b"):
		// Llama 3.1 405B: $2.00/$2.00 per million tokens (approximate)
		return PricingInfo{
			InputCostPerM:         2.00,
			OutputCostPerM:        2.00,
			CachedInputCostPerM:   2.00,
			CacheCreationCostPerM: 2.00,
		}
	case strings.Contains(lm, "mixtral"):
		// Mixtral 8x7B: $0.24/$0.24 per million tokens
		return PricingInfo{
			InputCostPerM:         0.24,
			OutputCostPerM:        0.24,
			CachedInputCostPerM:   0.24,
			CacheCreationCostPerM: 0.24,
		}
	case strings.Contains(lm, "gemma"):
		// Gemma 7B: $0.07/$0.07 per million tokens
		return PricingInfo{
			InputCostPerM:         0.07,
			OutputCostPerM:        0.07,
			CachedInputCostPerM:   0.07,
			CacheCreationCostPerM: 0.07,
		}
	case lm == "" || strings.Contains(lm, "llama-4-scout") || strings.Contains(lm, "8b"):
		// Llama 3/3.1 8B (default): $0.05/$0.10 per million tokens
		return PricingInfo{
			InputCostPerM:         0.05,
			OutputCostPerM:        0.10,
			CachedInputCostPerM:   0.05,
			CacheCreationCostPerM: 0.05,
		}
	default:
		return pricingUnavailableInfo()
	}
}
