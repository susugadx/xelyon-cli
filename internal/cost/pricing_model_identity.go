package cost

import "strings"

func pricingFamilyHasKnownModel(family, model string) bool {
	cfg := loadPricingConfig()
	if cfg == nil {
		return false
	}
	provider, ok := cfg.provider(family)
	if !ok {
		return false
	}
	return provider.hasKnownModel(model)
}

func (provider providerPricingConfig) hasKnownModel(model string) bool {
	model = normalizePricingModelName(model)
	if model == "" {
		return true
	}
	for _, exact := range provider.KnownModels.Exact {
		if normalizePricingModelName(exact) == model {
			return true
		}
	}
	return false
}

func normalizePricingModelName(model string) string {
	return strings.ToLower(strings.TrimSpace(model))
}
