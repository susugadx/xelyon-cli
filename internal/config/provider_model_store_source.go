package config

func (s providerModelStore) explicitSource(effective map[string]ProviderModelConfig) map[string]ProviderModelConfig {
	if s.state == providerModelSectionStateInMemoryEffectiveOnly {
		return rawProviderModelsFromEffectiveDiff(effective)
	}
	return s.raw
}

func (s providerModelStore) refreshSource(effective map[string]ProviderModelConfig) map[string]ProviderModelConfig {
	if s.state == providerModelSectionStateInMemoryEffectiveOnly {
		return effective
	}
	return s.raw
}

func (s providerModelStore) rawConfig(key string) (ProviderModelConfig, bool) {
	if key == "" || s.raw == nil {
		return ProviderModelConfig{}, false
	}
	pm, ok := s.raw[key]
	if !ok {
		return ProviderModelConfig{}, false
	}
	return cloneProviderModelConfig(pm), true
}
