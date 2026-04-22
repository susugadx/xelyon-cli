package config

import "gopkg.in/yaml.v3"

type providerModelStoreYAMLInput struct {
	sectionExists bool
	providerRaw   map[string]ProviderModelConfig
}

func extractRawProviderModelsFromYAML(data []byte) map[string]ProviderModelConfig {
	var raw struct {
		ProviderModels map[string]ProviderModelConfig `yaml:"provider_models"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil
	}
	return cloneProviderModelConfigMap(raw.ProviderModels)
}

func providerModelsSectionExists(raw map[string]interface{}) bool {
	return yamlRootHasKey(raw, "provider_models")
}

func providerModelStoreYAMLInputFromRoot(data []byte, raw map[string]interface{}) providerModelStoreYAMLInput {
	return providerModelStoreYAMLInput{
		sectionExists: providerModelsSectionExists(raw),
		providerRaw:   extractRawProviderModelsFromYAML(data),
	}
}
