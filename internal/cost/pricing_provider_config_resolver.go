package cost

import "strings"

func resolveProviderPricingFromConfig(provider providerPricingConfig, lm string, promptTokenCount int, allowLongInputTier bool) PricingInfo {
	if pricing, ok := matchPricingRules(lm, provider, promptTokenCount); ok {
		return pricing
	}
	if allowLongInputTier && provider.LongInput != nil && promptTokenCount > provider.LongInput.Threshold {
		return provider.LongInput.Pricing
	}
	return provider.Default
}

func resolveProviderPricingFromLoadedConfig(family string, lm string, promptTokenCount int, allowLongInputTier bool) (PricingInfo, bool) {
	cfg := loadPricingConfig()
	if cfg == nil {
		return PricingInfo{}, false
	}
	provider, ok := cfg.provider(family)
	if !ok {
		return PricingInfo{}, false
	}
	if !provider.hasKnownModel(lm) {
		return pricingUnavailableInfo(), true
	}
	return resolveKnownProviderPricingFromConfig(provider, lm, promptTokenCount, allowLongInputTier)
}

func resolveKnownProviderPricingFromConfig(provider providerPricingConfig, lm string, promptTokenCount int, allowLongInputTier bool) (PricingInfo, bool) {
	if pricing, ok := matchPricingRules(lm, provider, promptTokenCount); ok {
		return pricing, true
	}
	if strings.TrimSpace(lm) == "" {
		return resolveProviderPricingFromConfig(provider, lm, promptTokenCount, allowLongInputTier), true
	}
	return PricingInfo{}, false
}
