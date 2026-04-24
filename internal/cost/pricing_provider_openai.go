package cost

import (
	"strings"
)

// getOpenAIPricing はモデル名からOpenAI料金を返す
func getOpenAIPricing(model string, promptTokenCount int) PricingInfo {
	lm := strings.ToLower(model)
	if cfg := loadPricingConfig(); cfg != nil {
		return resolveProviderPricingFromConfig(cfg.OpenAI, lm, promptTokenCount, true)
	}

	switch {
	case strings.Contains(lm, "5.4-mini"):
		// GPT-5.4 Mini: $0.75/$4.50 per million tokens
		return PricingInfo{
			InputCostPerM:         0.75,
			OutputCostPerM:        4.50,
			CachedInputCostPerM:   0.075,
			CacheCreationCostPerM: 0.75,
		}
	case strings.Contains(lm, "5.4-nano"):
		// GPT-5.4 Nano: $0.20/$1.25 per million tokens
		return PricingInfo{
			InputCostPerM:         0.20,
			OutputCostPerM:        1.25,
			CachedInputCostPerM:   0.02,
			CacheCreationCostPerM: 0.20,
		}
	case strings.Contains(lm, "nano"):
		// GPT-5 Nano: $0.05/$0.40 per million tokens
		return PricingInfo{
			InputCostPerM:         0.05,
			OutputCostPerM:        0.40,
			CachedInputCostPerM:   0.005, // 90% off
			CacheCreationCostPerM: 0.05,
		}
	case strings.Contains(lm, "mini"):
		// GPT-5 Mini / Codex-Mini: $0.25/$2.00 per million tokens
		return PricingInfo{
			InputCostPerM:         0.25,
			OutputCostPerM:        2.00,
			CachedInputCostPerM:   0.025, // 90% off
			CacheCreationCostPerM: 0.25,
		}
	case strings.Contains(lm, "5.2-pro"):
		// GPT-5.2 Pro: $21/$168 per million tokens
		return PricingInfo{
			InputCostPerM:         21.00,
			OutputCostPerM:        168.00,
			CachedInputCostPerM:   2.10, // 90% off
			CacheCreationCostPerM: 21.00,
		}
	case strings.Contains(lm, "5.4-pro"):
		// GPT-5.4 Pro: $30/$180 per million tokens, >272K input doubles input-side pricing
		if promptTokenCount > 272000 {
			return PricingInfo{
				InputCostPerM:         60.00,
				OutputCostPerM:        180.00,
				CachedInputCostPerM:   6.00,
				CacheCreationCostPerM: 60.00,
			}
		}
		return PricingInfo{
			InputCostPerM:         30.00,
			OutputCostPerM:        180.00,
			CachedInputCostPerM:   3.00,
			CacheCreationCostPerM: 30.00,
		}
	case strings.Contains(lm, "5.4"):
		// GPT-5.4: $2.50/$15 per million tokens, >272K input doubles input-side pricing
		if promptTokenCount > 272000 {
			return PricingInfo{
				InputCostPerM:         5.00,
				OutputCostPerM:        15.00,
				CachedInputCostPerM:   0.50,
				CacheCreationCostPerM: 5.00,
			}
		}
		return PricingInfo{
			InputCostPerM:         2.50,
			OutputCostPerM:        15.00,
			CachedInputCostPerM:   0.25,
			CacheCreationCostPerM: 2.50,
		}
	case strings.Contains(lm, "5.1"):
		// GPT-5.1 / 5.1-Codex: $2.00/$8.00 per million tokens
		return PricingInfo{
			InputCostPerM:         2.00,
			OutputCostPerM:        8.00,
			CachedInputCostPerM:   0.50,
			CacheCreationCostPerM: 2.00,
		}
	case strings.Contains(lm, "5.3"):
		// GPT-5.3 / 5.3-Codex: $1.75/$14.00 per million tokens
		return PricingInfo{
			InputCostPerM:         1.75,
			OutputCostPerM:        14.00,
			CachedInputCostPerM:   0.175,
			CacheCreationCostPerM: 1.75,
		}
	case strings.Contains(lm, "5.2"):
		// GPT-5.2 / 5.2-Codex: $1.75/$14 per million tokens
		return PricingInfo{
			InputCostPerM:         1.75,
			OutputCostPerM:        14.00,
			CachedInputCostPerM:   0.175, // 90% off
			CacheCreationCostPerM: 1.75,
		}
	default:
		// GPT-5 / 5.1 / Codex（デフォルト）: $1.25/$10 per million tokens
		return PricingInfo{
			InputCostPerM:         1.25,
			OutputCostPerM:        10.00,
			CachedInputCostPerM:   0.125, // 90% off
			CacheCreationCostPerM: 1.25,
		}
	}
}
