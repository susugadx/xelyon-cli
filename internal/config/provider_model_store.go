package config

func (s providerModelStore) rawForSave() map[string]ProviderModelConfig {
	switch s.state {
	case providerModelSectionStateExplicitEmpty:
		return map[string]ProviderModelConfig{}
	case providerModelSectionStateExplicitEntries, providerModelSectionStateExplicitEntriesPreserveEmpty, providerModelSectionStateImplicitEntries:
		return cloneProviderModelConfigMap(s.raw)
	default:
		return nil
	}
}

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

func (s providerModelStore) rawForEdit(effective map[string]ProviderModelConfig) map[string]ProviderModelConfig {
	switch s.state {
	case providerModelSectionStateAbsent:
		return nil
	case providerModelSectionStateExplicitEmpty:
		return map[string]ProviderModelConfig{}
	case providerModelSectionStateExplicitEntries, providerModelSectionStateExplicitEntriesPreserveEmpty, providerModelSectionStateImplicitEntries:
		return cloneProviderModelConfigMap(s.raw)
	case providerModelSectionStateInMemoryEffectiveOnly:
		return cloneProviderModelConfigMap(effective)
	default:
		return nil
	}
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
