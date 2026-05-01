package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// GetProviderCompressThresholdWithConfig は明示設定された provider/model ごとの圧縮閾値を返す。
func GetProviderCompressThresholdWithConfig(cfg *config.Config, provider string, model string) int {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	normalizedProvider := config.NormalizeProviderName(provider)
	canonicalProvider := config.CanonicalProviderName(provider)
	normalizedModel := strings.ToLower(strings.TrimSpace(model))
	catalogModel := strings.ToLower(strings.TrimSpace(cfg.ModelCatalogName(provider, model)))
	if cfg.Compression.ProviderThresholds != nil {
		for _, candidate := range providerThresholdLookupKeys(normalizedProvider, canonicalProvider) {
			for _, lookupModel := range modelThresholdLookupNames(normalizedModel, catalogModel) {
				if threshold, ok := lookupConfiguredModelThreshold(cfg.Compression.ProviderThresholds, candidate, lookupModel); ok {
					return threshold
				}
			}
			if threshold, ok := cfg.Compression.ProviderThresholds[candidate]; ok {
				return threshold
			}
		}
	}

	return 0
}

// GetProviderCompressThreshold は明示設定された絶対トークン閾値を返す。
// 0 を返した場合は provider/model override なし。
func GetProviderCompressThreshold(provider string, model string) int {
	return GetProviderCompressThresholdWithConfig(config.DefaultConfig(), provider, model)
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

func modelThresholdLookupNames(model, catalogModel string) []string {
	if catalogModel == "" || catalogModel == model {
		return []string{model}
	}
	return []string{model, catalogModel}
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
