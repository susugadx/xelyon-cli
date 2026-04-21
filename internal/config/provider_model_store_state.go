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

func (s providerModelStore) stateForEntryMutation() providerModelSectionState {
	switch s.state {
	case providerModelSectionStateExplicitEmpty, providerModelSectionStateExplicitEntriesPreserveEmpty:
		return providerModelSectionStateExplicitEntriesPreserveEmpty
	case providerModelSectionStateExplicitEntries:
		return providerModelSectionStateExplicitEntries
	default:
		return providerModelSectionStateImplicitEntries
	}
}

func (s providerModelStore) stateAfterDeletingAllEntries() providerModelSectionState {
	switch s.state {
	case providerModelSectionStateExplicitEmpty, providerModelSectionStateExplicitEntriesPreserveEmpty:
		return providerModelSectionStateExplicitEmpty
	default:
		return providerModelSectionStateAbsent
	}
}

func (s providerModelStore) stateAfterEditingEntries(entryCount int) providerModelSectionState {
	switch s.state {
	case providerModelSectionStateExplicitEmpty:
		if entryCount == 0 {
			return providerModelSectionStateExplicitEmpty
		}
		return providerModelSectionStateExplicitEntriesPreserveEmpty
	case providerModelSectionStateExplicitEntries:
		if entryCount == 0 {
			return providerModelSectionStateAbsent
		}
		return providerModelSectionStateExplicitEntries
	case providerModelSectionStateExplicitEntriesPreserveEmpty:
		if entryCount == 0 {
			return providerModelSectionStateExplicitEmpty
		}
		return providerModelSectionStateExplicitEntriesPreserveEmpty
	case providerModelSectionStateAbsent, providerModelSectionStateImplicitEntries:
		if entryCount == 0 {
			return providerModelSectionStateAbsent
		}
		return providerModelSectionStateImplicitEntries
	case providerModelSectionStateInMemoryEffectiveOnly:
		if entryCount == 0 {
			return providerModelSectionStateInMemoryEffectiveOnly
		}
		return providerModelSectionStateImplicitEntries
	default:
		return providerModelSectionStateAbsent
	}
}

func (s providerModelStore) nextStateForMutation(raw map[string]ProviderModelConfig) providerModelSectionState {
	if len(raw) == 0 {
		return s.stateAfterDeletingAllEntries()
	}
	return s.stateForEntryMutation()
}

type providerModelStoreEditTransition struct {
	state                  providerModelSectionState
	raw                    map[string]ProviderModelConfig
	resetInMemoryEffective bool
}

func (s providerModelStore) transitionAfterEditingEntries(raw map[string]ProviderModelConfig) providerModelStoreEditTransition {
	nextState := s.stateAfterEditingEntries(len(raw))
	transition := providerModelStoreEditTransition{
		state: nextState,
	}

	switch nextState {
	case providerModelSectionStateInMemoryEffectiveOnly:
		transition.resetInMemoryEffective = true
	case providerModelSectionStateAbsent, providerModelSectionStateExplicitEmpty:
		transition.raw = nil
	default:
		if len(raw) > 0 {
			transition.raw = raw
		}
	}

	return transition
}

func (s providerModelStore) transitionAfterRawMutation(raw map[string]ProviderModelConfig) providerModelStoreEditTransition {
	nextState := s.nextStateForMutation(raw)
	if len(raw) == 0 {
		return providerModelStoreEditTransition{
			state: nextState,
			raw:   nil,
		}
	}
	return providerModelStoreEditTransition{
		state: nextState,
		raw:   raw,
	}
}
