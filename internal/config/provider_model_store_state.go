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

func providerModelStateAfterEditingNoEntries(state providerModelSectionState) providerModelSectionState {
	switch state {
	case providerModelSectionStateExplicitEmpty, providerModelSectionStateExplicitEntriesPreserveEmpty:
		return providerModelSectionStateExplicitEmpty
	case providerModelSectionStateInMemoryEffectiveOnly:
		return providerModelSectionStateInMemoryEffectiveOnly
	default:
		return providerModelSectionStateAbsent
	}
}

func providerModelStateAfterEditingEntries(state providerModelSectionState) providerModelSectionState {
	switch state {
	case providerModelSectionStateExplicitEmpty, providerModelSectionStateExplicitEntriesPreserveEmpty:
		return providerModelSectionStateExplicitEntriesPreserveEmpty
	case providerModelSectionStateExplicitEntries:
		return providerModelSectionStateExplicitEntries
	default:
		return providerModelSectionStateImplicitEntries
	}
}

func (s providerModelStore) stateAfterEditingEntries(entryCount int) providerModelSectionState {
	if entryCount == 0 {
		return providerModelStateAfterEditingNoEntries(s.state)
	}
	return providerModelStateAfterEditingEntries(s.state)
}

func (s providerModelStore) nextStateForMutation(raw map[string]ProviderModelConfig) providerModelSectionState {
	if len(raw) == 0 {
		return s.stateAfterDeletingAllEntries()
	}
	return s.stateForEntryMutation()
}

func providerModelStatePersistsRawEntries(state providerModelSectionState) bool {
	switch state {
	case providerModelSectionStateExplicitEntries, providerModelSectionStateExplicitEntriesPreserveEmpty, providerModelSectionStateImplicitEntries:
		return true
	default:
		return false
	}
}

type providerModelStoreEditTransition struct {
	state                  providerModelSectionState
	raw                    map[string]ProviderModelConfig
	resetInMemoryEffective bool
}

func providerModelStoreTransition(nextState providerModelSectionState, raw map[string]ProviderModelConfig, allowResetInMemoryEffective bool) providerModelStoreEditTransition {
	transition := providerModelStoreEditTransition{
		state: nextState,
	}
	if allowResetInMemoryEffective && nextState == providerModelSectionStateInMemoryEffectiveOnly {
		transition.resetInMemoryEffective = true
		return transition
	}
	if !providerModelStatePersistsRawEntries(nextState) || len(raw) == 0 {
		return transition
	}
	transition.raw = raw
	return transition
}

func (s providerModelStore) transitionAfterEditingEntries(raw map[string]ProviderModelConfig) providerModelStoreEditTransition {
	nextState := s.stateAfterEditingEntries(len(raw))
	return providerModelStoreTransition(nextState, raw, true)
}

func (s providerModelStore) transitionAfterRawMutation(raw map[string]ProviderModelConfig) providerModelStoreEditTransition {
	nextState := s.nextStateForMutation(raw)
	return providerModelStoreTransition(nextState, raw, false)
}
