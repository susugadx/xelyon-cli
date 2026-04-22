package config

import "gopkg.in/yaml.v3"

func extractRawProviderModelsFromYAML(data []byte) map[string]ProviderModelConfig {
	var raw struct {
		ProviderModels map[string]ProviderModelConfig `yaml:"provider_models"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil
	}
	return cloneProviderModelConfigMap(raw.ProviderModels)
}

func providerModelStoreStateFromParsedYAML(providerModelsRaw map[string]ProviderModelConfig) providerModelStore {
	if len(providerModelsRaw) == 0 {
		return normalizeProviderModelStore(providerModelSectionStateExplicitEmpty, nil)
	}
	return normalizeProviderModelStore(providerModelSectionStateExplicitEntries, providerModelsRaw)
}

func providerModelsSectionExists(raw map[string]interface{}) bool {
	return yamlRootHasKey(raw, "provider_models")
}

func providerModelStoreStateFromYAMLSection(sectionExists bool, providerModelsRaw map[string]ProviderModelConfig) providerModelStore {
	if !sectionExists {
		return normalizeProviderModelStore(providerModelSectionStateAbsent, nil)
	}
	return providerModelStoreStateFromParsedYAML(providerModelsRaw)
}

func providerModelStoreFromYAMLWithRoot(data []byte, raw map[string]interface{}) providerModelStore {
	return providerModelStoreStateFromYAMLSection(
		providerModelsSectionExists(raw),
		extractRawProviderModelsFromYAML(data),
	)
}
