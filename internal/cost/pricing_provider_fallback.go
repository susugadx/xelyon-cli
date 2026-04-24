package cost

func getUnknownProviderFallbackPricing() PricingInfo {
	// 不明なプロバイダーはDeepSeek V3.2料金で概算
	return PricingInfo{
		InputCostPerM:         0.28,
		OutputCostPerM:        0.42,
		CachedInputCostPerM:   0.028,
		CacheCreationCostPerM: 0.28,
	}
}
