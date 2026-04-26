package cost

import (
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// CalculateRequestCost は単一リクエストのコストを計算（キャッシュなし想定）
func CalculateRequestCost(provider, model string, input, output int) float64 {
	return CalculateRequestCostForConfig(nil, provider, model, input, output)
}

// CalculateRequestCostForConfig は catalog_model 設定を考慮して単一リクエストのコストを計算する。
func CalculateRequestCostForConfig(cfg *config.Config, provider, model string, input, output int) float64 {
	if provider == "ollama" {
		return 0.0 // ローカル実行
	}

	pricing := GetPricingInfoForConfig(cfg, provider, model, input)

	// コスト計算: (tokens / 1,000,000) * price
	inputCostUSD := (float64(input) / 1_000_000.0) * pricing.InputCostPerM
	outputCostUSD := (float64(output) / 1_000_000.0) * pricing.OutputCostPerM

	return inputCostUSD + outputCostUSD
}

// CalculateRequestCostWithCache は単一リクエストのコストを計算（キャッシュ対応）
func CalculateRequestCostWithCache(provider, model string, usage api.Usage) float64 {
	return CalculateRequestCostWithCacheForConfig(nil, provider, model, usage)
}

// CalculateRequestCostWithCacheForConfig は catalog_model 設定を考慮してキャッシュ対応コストを計算する。
func CalculateRequestCostWithCacheForConfig(cfg *config.Config, provider, model string, usage api.Usage) float64 {
	if provider == "ollama" {
		return 0.0
	}

	tierInputTokens := pricingTierInputTokensForUsage(cfg, provider, model, usage)
	pricing := GetPricingInfoForConfig(cfg, provider, model, tierInputTokens)

	cachedInputCost := float64(usage.CachedInputTokens) / 1_000_000.0 * pricing.CachedInputCostPerM
	cacheCreationCost := float64(usage.CacheCreationTokens) / 1_000_000.0 * pricing.CacheCreationCostPerM

	uncachedInput := usage.InputTokens - usage.CachedInputTokens - usage.CacheCreationTokens
	if uncachedInput < 0 {
		uncachedInput = 0
	}
	uncachedInputCost := float64(uncachedInput) / 1_000_000.0 * pricing.InputCostPerM

	outputCost := float64(usage.OutputTokens) / 1_000_000.0 * pricing.OutputCostPerM

	// Thinking tokens は出力レートで課金（candidatesTokenCount とは別計上）
	thinkingCost := float64(usage.ThinkingTokens) / 1_000_000.0 * pricing.OutputCostPerM

	return cachedInputCost + cacheCreationCost + uncachedInputCost + outputCost + thinkingCost
}
