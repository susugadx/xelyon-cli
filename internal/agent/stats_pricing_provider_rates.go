package agent

import (
	"strings"
)

// getDeepSeekPricing はモデル名からDeepSeek料金を返す
func getDeepSeekPricing(model string) PricingInfo {
	lm := strings.ToLower(model)
	if cfg := loadPricingConfig(); cfg != nil {
		provider := cfg.DeepSeek
		if pricing, ok := matchPricingRules(lm, provider, 0); ok {
			return pricing
		}
		return provider.Default
	}

	// DeepSeek V3.2: deepseek-chat/deepseek-reasoner 統一料金
	// $0.28/$0.42 per million tokens, Cache hit: $0.028
	return PricingInfo{
		InputCostPerM:         0.28,
		OutputCostPerM:        0.42,
		CachedInputCostPerM:   0.028,
		CacheCreationCostPerM: 0.28,
	}
}

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

// getOpenAIPricing はモデル名からOpenAI料金を返す
func getOpenAIPricing(model string, promptTokenCount int) PricingInfo {
	lm := strings.ToLower(model)
	if cfg := loadPricingConfig(); cfg != nil {
		provider := cfg.OpenAI
		if pricing, ok := matchPricingRules(lm, provider, promptTokenCount); ok {
			return pricing
		}
		if provider.LongInput != nil && promptTokenCount > provider.LongInput.Threshold {
			return provider.LongInput.Pricing
		}
		return provider.Default
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

// getGeminiPricing はモデル名からGemini料金を返す
// promptTokenCount はリクエストの入力トークン数（200Kティア判定に使用）
func getGeminiPricing(model string, promptTokenCount int) PricingInfo {
	lm := strings.ToLower(model)
	if cfg := loadPricingConfig(); cfg != nil {
		provider := cfg.Gemini
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
	default:
		// Gemini 3 Flash（デフォルト）: $0.50/$3.00 per million tokens
		return PricingInfo{
			InputCostPerM: 0.50, OutputCostPerM: 3.00,
			CachedInputCostPerM: 0.05, CacheCreationCostPerM: 0.50,
		}
	}
}

// getGroqPricing はモデル名からGroq料金を返す
func getGroqPricing(model string) PricingInfo {
	lm := strings.ToLower(model)
	if cfg := loadPricingConfig(); cfg != nil {
		provider := cfg.Groq
		if pricing, ok := matchPricingRules(lm, provider, 0); ok {
			return pricing
		}
		return provider.Default
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
	default:
		// Llama 3/3.1 8B (default): $0.05/$0.10 per million tokens
		return PricingInfo{
			InputCostPerM:         0.05,
			OutputCostPerM:        0.10,
			CachedInputCostPerM:   0.05,
			CacheCreationCostPerM: 0.05,
		}
	}
}

// getKimiPricing はモデル名からKimi料金を返す
func getKimiPricing(model string) PricingInfo {
	lm := strings.ToLower(model)
	if cfg := loadPricingConfig(); cfg != nil {
		provider := cfg.Kimi
		if pricing, ok := matchPricingRules(lm, provider, 0); ok {
			return pricing
		}
		return provider.Default
	}

	if strings.Contains(lm, "k2.5") {
		// Kimi K2.5: $0.60/$3.00 per million tokens
		return PricingInfo{
			InputCostPerM:         0.60,
			OutputCostPerM:        3.00,
			CachedInputCostPerM:   0.06,
			CacheCreationCostPerM: 0.60,
		}
	}
	// Kimi K2（デフォルト）: $0.60/$2.50 per million tokens
	return PricingInfo{
		InputCostPerM:         0.60,
		OutputCostPerM:        2.50,
		CachedInputCostPerM:   0.06,
		CacheCreationCostPerM: 0.60,
	}
}
