package cost

import (
	"github.com/susugadx/xelyon-cli/internal/api"
)

// CalculateRequestCost は単一リクエストのコストを計算（キャッシュなし想定）
func CalculateRequestCost(provider, model string, input, output int) float64 {
	if provider == "ollama" {
		return 0.0 // ローカル実行
	}

	pricing := GetPricingInfo(provider, model, input)

	// コスト計算: (tokens / 1,000,000) * price
	inputCostUSD := (float64(input) / 1_000_000.0) * pricing.InputCostPerM
	outputCostUSD := (float64(output) / 1_000_000.0) * pricing.OutputCostPerM

	return inputCostUSD + outputCostUSD
}

// CalculateRequestCostWithCache は単一リクエストのコストを計算（キャッシュ対応）
func CalculateRequestCostWithCache(provider, model string, usage api.Usage) float64 {
	if provider == "ollama" {
		return 0.0
	}

	// キャッシュトークンも含めた総入力トークンで200Kティア判定
	// Anthropic/Gemini公式: input + cache_read + cache_creation の合計でティア判定
	totalInputForTier := usage.InputTokens + usage.CachedInputTokens + usage.CacheCreationTokens
	pricing := GetPricingInfo(provider, model, totalInputForTier)

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
