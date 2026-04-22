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
