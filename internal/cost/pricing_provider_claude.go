package cost

import (
	"strings"
)

// getClaudePricing はモデル名からClaude料金を返す。
// Claude API の Fable 5 / Opus 4.8 / Opus 4.7 / Opus 4.6 / Sonnet 4.6 は 1M context でも標準単価。
func getClaudePricing(model string, promptTokenCount int) PricingInfo {
	lm := strings.ToLower(model)
	if pricing, ok := resolveProviderPricingFromLoadedConfig("claude", lm, promptTokenCount, true); ok {
		return pricing
	}

	switch {
	case strings.Contains(lm, "fable"):
		return PricingInfo{
			InputCostPerM:         10.00,
			OutputCostPerM:        50.00,
			CachedInputCostPerM:   1.00,
			CacheCreationCostPerM: 12.50,
		}
	case strings.Contains(lm, "opus"):
		if promptTokenCount > 200000 && !claudeOpusUsesStandardLongContextPricing(lm) {
			return PricingInfo{
				InputCostPerM:         10.00,
				OutputCostPerM:        37.50,
				CachedInputCostPerM:   1.00,
				CacheCreationCostPerM: 12.50,
			}
		}
		return PricingInfo{
			InputCostPerM:         5.00,
			OutputCostPerM:        25.00,
			CachedInputCostPerM:   0.50, // 90% off
			CacheCreationCostPerM: 6.25, // 25% premium
		}
	case strings.Contains(lm, "haiku"):
		// Haiku 4.5: $1/$5 per million tokens (long context なし)
		return PricingInfo{
			InputCostPerM:         1.00,
			OutputCostPerM:        5.00,
			CachedInputCostPerM:   0.10, // 90% off
			CacheCreationCostPerM: 1.25, // 25% premium
		}
	case lm == "":
		return PricingInfo{
			InputCostPerM:         3.00,
			OutputCostPerM:        15.00,
			CachedInputCostPerM:   0.30, // 90% off
			CacheCreationCostPerM: 3.75, // 25% premium
		}
	case strings.Contains(lm, "sonnet"):
		if promptTokenCount > 200000 && !claudeSonnetUsesStandardLongContextPricing(lm) {
			return PricingInfo{
				InputCostPerM:         6.00,
				OutputCostPerM:        22.50,
				CachedInputCostPerM:   0.60,
				CacheCreationCostPerM: 7.50,
			}
		}
		return PricingInfo{
			InputCostPerM:         3.00,
			OutputCostPerM:        15.00,
			CachedInputCostPerM:   0.30, // 90% off
			CacheCreationCostPerM: 3.75, // 25% premium
		}
	default:
		return pricingUnavailableInfo()
	}
}

func claudeOpusUsesStandardLongContextPricing(lm string) bool {
	return strings.Contains(lm, "opus-4-8") ||
		strings.Contains(lm, "opus-4.8") ||
		strings.Contains(lm, "opus-4-7") ||
		strings.Contains(lm, "opus-4.7") ||
		strings.Contains(lm, "opus-4-6") ||
		strings.Contains(lm, "opus-4.6")
}

func claudeSonnetUsesStandardLongContextPricing(lm string) bool {
	return strings.Contains(lm, "sonnet-4-6") ||
		strings.Contains(lm, "sonnet-4.6")
}
