package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

// GetProviderCompressThresholdWithConfig は設定を考慮して provider/model ごとの圧縮閾値を返す。
func GetProviderCompressThresholdWithConfig(cfg *config.Config, provider string, model string) int {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	normalizedProvider := config.NormalizeProviderName(provider)
	canonicalProvider := config.CanonicalProviderName(provider)
	normalizedModel := strings.ToLower(strings.TrimSpace(model))
	if cfg.Compression.ProviderThresholds != nil {
		for _, candidate := range providerThresholdLookupKeys(normalizedProvider, canonicalProvider) {
			if threshold, ok := lookupConfiguredModelThreshold(cfg.Compression.ProviderThresholds, candidate, normalizedModel); ok {
				return threshold
			}
			if threshold, ok := cfg.Compression.ProviderThresholds[candidate]; ok {
				return threshold
			}
		}
	}

	return defaultProviderCompressThreshold(canonicalProvider, normalizedModel)
}

// GetProviderCompressThreshold はプロバイダとモデルに基づく
// コスト最適化のための絶対トークン閾値を返す。
// 0 を返した場合は絶対値閾値なし（既存の%ベースを使用）。
func GetProviderCompressThreshold(provider string, model string) int {
	return GetProviderCompressThresholdWithConfig(config.DefaultConfig(), provider, model)
}

func defaultProviderCompressThreshold(provider string, model string) int {
	entry, ok := llmcatalog.ProviderDescriptorFor(provider)
	if !ok {
		return 0
	}
	return entry.CompressionThreshold
}

func providerThresholdLookupKeys(normalizedProvider, canonicalProvider string) []string {
	keys := config.ProviderModelLookupKeys(normalizedProvider)
	if len(keys) > 0 {
		return keys
	}
	if canonicalProvider == "" {
		return nil
	}
	return []string{canonicalProvider}
}

func lookupConfiguredModelThreshold(thresholds map[string]int, provider string, model string) (int, bool) {
	if len(thresholds) == 0 || provider == "" || model == "" {
		return 0, false
	}

	if threshold, ok := thresholds[provider+":"+model]; ok {
		return threshold, true
	}

	bestLen := -1
	bestValue := 0
	prefix := provider + ":"
	for key, threshold := range thresholds {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		pattern := strings.TrimPrefix(key, prefix)
		if pattern == "" || pattern == model {
			continue
		}

		matchPrefix := strings.TrimSuffix(pattern, "*")
		if matchPrefix == "" || !strings.HasPrefix(model, matchPrefix) {
			continue
		}

		if len(matchPrefix) > bestLen {
			bestLen = len(matchPrefix)
			bestValue = threshold
		}
	}

	if bestLen >= 0 {
		return bestValue, true
	}
	return 0, false
}

func averageOutputTokens(stats *SessionStats) int {
	if stats == nil || stats.OutputTokens <= 0 {
		return 0
	}
	assistantMessages := stats.AssistantMessages
	if assistantMessages < 1 {
		assistantMessages = 1
	}
	return stats.OutputTokens / assistantMessages
}

func shouldForceCompressForPricingCliff(provider, model string, currentTokens int, stats *SessionStats) (int, bool) {
	if currentTokens <= 0 {
		return currentTokens, false
	}

	projectedTokens := currentTokens + averageOutputTokens(stats)
	currentPricing := cost.GetPricingInfo(provider, model, currentTokens)
	projectedPricing := cost.GetPricingInfo(provider, model, projectedTokens)
	if projectedPricing.InputCostPerM > currentPricing.InputCostPerM {
		return projectedTokens, true
	}

	return projectedTokens, false
}
