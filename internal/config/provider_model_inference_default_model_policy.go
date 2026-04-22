package config

import "strings"

func (c *Config) configuredDefaultModelAppliesToProvider(provider, model string) bool {
	if model == "" {
		return false
	}

	if c != nil {
		matchedAnyProvider := false
		for _, candidate := range providerConfigCandidates() {
			pm, ok := c.GetProviderModelConfig(candidate)
			if !ok || pm.DefaultModel != model {
				continue
			}
			matchedAnyProvider = true
			if SameProviderRuntimeIdentity(provider, candidate) {
				return true
			}
		}
		if matchedAnyProvider {
			return false
		}
	}

	if providerTreatsConfiguredDefaultModelAsOpaque(provider, model) {
		return true
	}

	inferredProvider := InferProviderFromModel(model)
	if inferredProvider == "" {
		return true
	}
	return SameProviderRuntimeIdentity(provider, inferredProvider)
}

func providerTreatsConfiguredDefaultModelAsOpaque(provider, model string) bool {
	switch CanonicalProviderName(provider) {
	case "ollama":
		return true
	case "groq":
		return strings.Contains(strings.TrimSpace(model), "/")
	default:
		return false
	}
}
