package cost

func resolveProviderPricingFromConfig(provider providerPricingConfig, lm string, promptTokenCount int, allowLongInputTier bool) PricingInfo {
	if pricing, ok := matchPricingRules(lm, provider, promptTokenCount); ok {
		return pricing
	}
	if allowLongInputTier && provider.LongInput != nil && promptTokenCount > provider.LongInput.Threshold {
		return provider.LongInput.Pricing
	}
	return provider.Default
}
