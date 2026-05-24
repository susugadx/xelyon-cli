package cost

import (
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// CostEstimate はUSD見積もりと、その見積もりが価格表で完結しているかを表す。
// PricingUnavailable が true の場合、Cost は既知部分だけで総額として表示してはいけない。
type CostEstimate struct {
	Cost               float64
	PricingUnavailable bool
}

// CalculateRequestCost は単一リクエストのコストを計算（キャッシュなし想定）
func CalculateRequestCost(provider, model string, input, output int) float64 {
	return CalculateRequestCostForConfig(nil, provider, model, input, output)
}

// CalculateRequestCostForConfig は catalog_model 設定を考慮して単一リクエストのコストを計算する。
func CalculateRequestCostForConfig(cfg *config.Config, provider, model string, input, output int) float64 {
	return EstimateRequestCostForConfig(cfg, provider, model, input, output).Cost
}

func EstimateRequestCost(provider, model string, input, output int) CostEstimate {
	return EstimateRequestCostForConfig(nil, provider, model, input, output)
}

func EstimateRequestCostForConfig(cfg *config.Config, provider, model string, input, output int) CostEstimate {
	if provider == "ollama" {
		return CostEstimate{} // ローカル実行
	}

	pricing := GetPricingInfoForConfig(cfg, provider, model, input)
	if pricing.PricingUnavailable {
		return CostEstimate{PricingUnavailable: true}
	}

	// コスト計算: (tokens / 1,000,000) * price
	inputCostUSD := (float64(input) / 1_000_000.0) * pricing.InputCostPerM
	outputCostUSD := (float64(output) / 1_000_000.0) * pricing.OutputCostPerM

	return CostEstimate{Cost: inputCostUSD + outputCostUSD}
}

// CalculateRequestCostWithCache は単一リクエストのコストを計算（キャッシュ対応）
func CalculateRequestCostWithCache(provider, model string, usage api.Usage) float64 {
	return CalculateRequestCostWithCacheForConfig(nil, provider, model, usage)
}

// CalculateRequestCostWithCacheForConfig は catalog_model 設定を考慮してキャッシュ対応コストを計算する。
func CalculateRequestCostWithCacheForConfig(cfg *config.Config, provider, model string, usage api.Usage) float64 {
	return EstimateRequestCostWithCacheForConfig(cfg, provider, model, usage).Cost
}

func EstimateRequestCostWithCache(provider, model string, usage api.Usage) CostEstimate {
	return EstimateRequestCostWithCacheForConfig(nil, provider, model, usage)
}

func EstimateRequestCostWithCacheForConfig(cfg *config.Config, provider, model string, usage api.Usage) CostEstimate {
	if provider == "ollama" {
		return CostEstimate{}
	}

	tierInputTokens := pricingTierInputTokensForUsage(cfg, provider, model, usage)
	pricing := GetPricingInfoForConfig(cfg, provider, model, tierInputTokens)
	if pricing.PricingUnavailable {
		return CostEstimate{PricingUnavailable: true}
	}

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

	return CostEstimate{Cost: cachedInputCost + cacheCreationCost + uncachedInputCost + outputCost + thinkingCost}
}

// EstimateCacheStorageCost は explicit context cache の保管コストを計算する。
func EstimateCacheStorageCost(provider, model string, tokens, ttlSeconds int) CostEstimate {
	return EstimateCacheStorageCostForConfig(nil, provider, model, tokens, ttlSeconds)
}

// EstimateCacheStorageCostForConfig は catalog_model 設定を考慮して explicit context cache の保管コストを計算する。
func EstimateCacheStorageCostForConfig(cfg *config.Config, provider, model string, tokens, ttlSeconds int) CostEstimate {
	if provider == "ollama" || tokens <= 0 || ttlSeconds <= 0 {
		return CostEstimate{}
	}

	pricing := GetPricingInfoForConfig(cfg, provider, model, tokens)
	if pricing.PricingUnavailable {
		return CostEstimate{PricingUnavailable: true}
	}
	if pricing.CacheStorageCostPerMHour <= 0 {
		return CostEstimate{}
	}

	ttlHours := float64(ttlSeconds) / 3600.0
	cost := float64(tokens) / 1_000_000.0 * pricing.CacheStorageCostPerMHour * ttlHours
	return CostEstimate{Cost: cost}
}
