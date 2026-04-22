package config

func providerModelStoreStateFromParsedYAML(providerModelsRaw map[string]ProviderModelConfig) providerModelStore {
	if len(providerModelsRaw) == 0 {
		return normalizeProviderModelStore(providerModelSectionStateExplicitEmpty, nil)
	}
	return normalizeProviderModelStore(providerModelSectionStateExplicitEntries, providerModelsRaw)
}

func providerModelStoreStateFromYAMLSection(sectionExists bool, providerModelsRaw map[string]ProviderModelConfig) providerModelStore {
	if !sectionExists {
		return normalizeProviderModelStore(providerModelSectionStateAbsent, nil)
	}
	return providerModelStoreStateFromParsedYAML(providerModelsRaw)
}

func providerModelStoreFromYAMLInput(input providerModelStoreYAMLInput) providerModelStore {
	return providerModelStoreStateFromYAMLSection(input.sectionExists, input.providerRaw)
}
