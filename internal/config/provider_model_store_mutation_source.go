package config

func (s providerModelStore) clonedRawForMutation(effective map[string]ProviderModelConfig) map[string]ProviderModelConfig {
	raw := cloneProviderModelConfigMap(s.raw)
	if raw == nil && s.state == providerModelSectionStateInMemoryEffectiveOnly {
		raw = rawProviderModelsFromEffectiveDiff(effective)
	}
	return raw
}

func (s providerModelStore) mutableRawForMutation(effective map[string]ProviderModelConfig) map[string]ProviderModelConfig {
	raw := s.clonedRawForMutation(effective)
	if raw == nil {
		raw = map[string]ProviderModelConfig{}
	}
	return raw
}
