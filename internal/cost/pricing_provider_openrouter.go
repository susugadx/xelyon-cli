package cost

func getOpenRouterPricing(model string, promptTokenCount int) PricingInfo {
	if pricing, ok := resolveOpenRouterDelegatedProviderPricing(model, promptTokenCount); ok {
		return pricing
	}
	if pricing, ok := resolveOpenRouterStaticPricing(model); ok {
		return pricing
	}
	return pricingUnavailableInfo()
}
