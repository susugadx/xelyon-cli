package agent

import (
	"strings"
)

// getClaudePricing はモデル名からClaude料金を返す
// promptTokenCount は200Kティア判定に使用（Geminiと同様）
func getClaudePricing(model string, promptTokenCount int) PricingInfo {
	lm := strings.ToLower(model)
	if cfg := loadPricingConfig(); cfg != nil {
		provider := cfg.Claude
		if pricing, ok := matchPricingRules(lm, provider, promptTokenCount); ok {
			return pricing
		}
		// デフォルトの long_input チェック
		if provider.LongInput != nil && promptTokenCount > provider.LongInput.Threshold {
			return provider.LongInput.Pricing
		}
		return provider.Default
	}

	switch {
	case strings.Contains(lm, "opus"):
		if promptTokenCount > 200000 {
			// Opus long context (>200K): $10/$37.50 per million tokens
			return PricingInfo{
				InputCostPerM:         10.00,
				OutputCostPerM:        37.50,
				CachedInputCostPerM:   1.00,
				CacheCreationCostPerM: 12.50,
			}
		}
		// Opus 4.5/4.6: $5/$25 per million tokens
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
	default:
		if promptTokenCount > 200000 {
			// Sonnet long context (>200K): $6/$22.50 per million tokens
			return PricingInfo{
				InputCostPerM:         6.00,
				OutputCostPerM:        22.50,
				CachedInputCostPerM:   0.60,
				CacheCreationCostPerM: 7.50,
			}
		}
		// Sonnet 4.5/4.6（デフォルト）: $3/$15 per million tokens
		return PricingInfo{
			InputCostPerM:         3.00,
			OutputCostPerM:        15.00,
			CachedInputCostPerM:   0.30, // 90% off
			CacheCreationCostPerM: 3.75, // 25% premium
		}
	}
}
