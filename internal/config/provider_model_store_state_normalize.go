package config

func normalizeProviderModelStore(state providerModelSectionState, raw map[string]ProviderModelConfig) providerModelStore {
	cloned := cloneProviderModelConfigMap(raw)

	switch state {
	case providerModelSectionStateAbsent, providerModelSectionStateInMemoryEffectiveOnly:
		cloned = nil
	case providerModelSectionStateExplicitEmpty:
		cloned = map[string]ProviderModelConfig{}
	case providerModelSectionStateExplicitEntries:
		if len(cloned) == 0 {
			state = providerModelSectionStateExplicitEmpty
			cloned = map[string]ProviderModelConfig{}
		}
	case providerModelSectionStateExplicitEntriesPreserveEmpty:
		if len(cloned) == 0 {
			state = providerModelSectionStateExplicitEmpty
			cloned = map[string]ProviderModelConfig{}
		}
	case providerModelSectionStateImplicitEntries:
		if len(cloned) == 0 {
			state = providerModelSectionStateAbsent
			cloned = nil
		}
	default:
		state = providerModelSectionStateAbsent
		cloned = nil
	}

	return providerModelStore{
		state: state,
		raw:   cloned,
	}
}

func (s providerModelStore) clone() providerModelStore {
	return normalizeProviderModelStore(s.state, s.raw)
}
