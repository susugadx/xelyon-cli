package config

import (
	"reflect"

	"gopkg.in/yaml.v3"
)

func cloneModelOverrides(src map[string]ModelOverride) map[string]ModelOverride {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string]ModelOverride, len(src))
	for key, value := range src {
		cloned[key] = value
	}
	return cloned
}

func cloneProviderModelConfig(src ProviderModelConfig) ProviderModelConfig {
	cloned := src
	cloned.AnthropicBeta = append([]string(nil), src.AnthropicBeta...)
	cloned.ModelOverrides = cloneModelOverrides(src.ModelOverrides)
	return cloned
}

func cloneProviderModelConfigMap(src map[string]ProviderModelConfig) map[string]ProviderModelConfig {
	if src == nil {
		return nil
	}
	cloned := make(map[string]ProviderModelConfig, len(src))
	for key, value := range src {
		normalized := NormalizeProviderName(key)
		if normalized == "" {
			continue
		}
		cloned[normalized] = cloneProviderModelConfig(value)
	}
	return cloned
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

func providerModelStoreFromYAML(data []byte) providerModelStore {
	if !yamlHasKey(data, "provider_models") {
		return normalizeProviderModelStore(providerModelSectionStateAbsent, nil)
	}

	raw := extractRawProviderModelsFromYAML(data)
	if len(raw) == 0 {
		return normalizeProviderModelStore(providerModelSectionStateExplicitEmpty, nil)
	}

	return normalizeProviderModelStore(providerModelSectionStateExplicitEntries, raw)
}

func mergeProviderModelConfig(base, override ProviderModelConfig) ProviderModelConfig {
	merged := base

	if override.DefaultModel != "" {
		merged.DefaultModel = override.DefaultModel
	}
	if override.MaxOutputTokens > 0 {
		merged.MaxOutputTokens = override.MaxOutputTokens
	}
	if override.AnthropicVersion != "" {
		merged.AnthropicVersion = override.AnthropicVersion
	}
	if len(override.AnthropicBeta) > 0 {
		merged.AnthropicBeta = append([]string(nil), override.AnthropicBeta...)
	}

	if len(base.ModelOverrides) == 0 && len(override.ModelOverrides) == 0 {
		merged.ModelOverrides = nil
		return merged
	}

	merged.ModelOverrides = cloneModelOverrides(base.ModelOverrides)
	if merged.ModelOverrides == nil {
		merged.ModelOverrides = map[string]ModelOverride{}
	}
	for key, value := range override.ModelOverrides {
		merged.ModelOverrides[key] = value
	}

	return merged
}

func isZeroProviderModelConfig(pm ProviderModelConfig) bool {
	return pm.DefaultModel == "" &&
		pm.MaxOutputTokens == 0 &&
		pm.AnthropicVersion == "" &&
		len(pm.AnthropicBeta) == 0 &&
		len(pm.ModelOverrides) == 0
}

func diffProviderModelConfig(base, current ProviderModelConfig) ProviderModelConfig {
	diff := ProviderModelConfig{}

	if current.DefaultModel != "" && current.DefaultModel != base.DefaultModel {
		diff.DefaultModel = current.DefaultModel
	}
	if current.MaxOutputTokens > 0 && current.MaxOutputTokens != base.MaxOutputTokens {
		diff.MaxOutputTokens = current.MaxOutputTokens
	}
	if current.AnthropicVersion != "" && current.AnthropicVersion != base.AnthropicVersion {
		diff.AnthropicVersion = current.AnthropicVersion
	}
	if len(current.AnthropicBeta) > 0 && !reflect.DeepEqual(current.AnthropicBeta, base.AnthropicBeta) {
		diff.AnthropicBeta = append([]string(nil), current.AnthropicBeta...)
	}
	if len(current.ModelOverrides) > 0 && !reflect.DeepEqual(current.ModelOverrides, base.ModelOverrides) {
		diff.ModelOverrides = cloneModelOverrides(current.ModelOverrides)
	}

	return diff
}

func rawProviderModelsFromEffectiveDiff(effective map[string]ProviderModelConfig) map[string]ProviderModelConfig {
	if len(effective) == 0 {
		return nil
	}

	raw := map[string]ProviderModelConfig{}
	for key, pm := range effective {
		normalized := NormalizeProviderName(key)
		if normalized == "" {
			continue
		}

		base, ok := defaultProviderModelConfig(normalized)
		if ok {
			pm = diffProviderModelConfig(base, pm)
		}
		if isZeroProviderModelConfig(pm) {
			continue
		}
		raw[normalized] = cloneProviderModelConfig(pm)
	}

	if len(raw) == 0 {
		return nil
	}
	return raw
}

func buildEffectiveProviderModels(src map[string]ProviderModelConfig) map[string]ProviderModelConfig {
	effective := cloneProviderModelConfigMap(DefaultConfig().ProviderModels)
	if effective == nil {
		effective = map[string]ProviderModelConfig{}
	}

	for key, pm := range src {
		base, ok := defaultProviderModelConfig(key)
		if ok {
			effective[key] = mergeProviderModelConfig(base, pm)
			continue
		}
		effective[key] = cloneProviderModelConfig(pm)
	}

	return effective
}

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

func (s providerModelStore) nextStateForMutation(raw map[string]ProviderModelConfig) providerModelSectionState {
	if len(raw) == 0 {
		return s.stateAfterDeletingAllEntries()
	}
	return s.stateForEntryMutation()
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

func (c *Config) effectiveProviderModels() map[string]ProviderModelConfig {
	if c == nil {
		return nil
	}
	return c.ProviderModels
}

func (c *Config) effectiveProviderModelConfig(key string) (ProviderModelConfig, bool) {
	if c == nil || key == "" || c.ProviderModels == nil {
		return ProviderModelConfig{}, false
	}
	pm, ok := c.ProviderModels[key]
	if !ok {
		return ProviderModelConfig{}, false
	}
	return cloneProviderModelConfig(pm), true
}

func (c *Config) setEffectiveProviderModelConfig(key string, pm ProviderModelConfig) {
	if c == nil || key == "" {
		return
	}
	if c.ProviderModels == nil {
		c.ProviderModels = map[string]ProviderModelConfig{}
	}
	c.ProviderModels[key] = cloneProviderModelConfig(pm)
}

func (c *Config) deleteEffectiveProviderModelConfig(key string) {
	if c == nil || key == "" || c.ProviderModels == nil {
		return
	}
	delete(c.ProviderModels, key)
}

func (c *Config) effectiveProviderModelRefreshSource() map[string]ProviderModelConfig {
	if c == nil {
		return nil
	}
	return c.providerModelsStore.refreshSource(c.effectiveProviderModels())
}

func (c *Config) refreshEffectiveProviderModels() {
	if c == nil {
		return
	}
	c.ProviderModels = buildEffectiveProviderModels(c.effectiveProviderModelRefreshSource())
}

func (c *Config) setProviderModelStoreState(state providerModelSectionState, raw map[string]ProviderModelConfig) {
	if c == nil {
		return
	}
	c.providerModelsStore = normalizeProviderModelStore(state, raw)
	c.refreshEffectiveProviderModels()
}

func (c *Config) resetInMemoryEffectiveProviderModels() {
	if c == nil {
		return
	}
	c.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateInMemoryEffectiveOnly, nil)
	c.ProviderModels = buildEffectiveProviderModels(nil)
}

func (c *Config) clonedRawProviderModelsForMutation() map[string]ProviderModelConfig {
	if c == nil {
		return nil
	}
	return c.providerModelsStore.clonedRawForMutation(c.effectiveProviderModels())
}

func (c *Config) mutableRawProviderModelsForMutation() map[string]ProviderModelConfig {
	if c == nil {
		return nil
	}
	return c.providerModelsStore.mutableRawForMutation(c.effectiveProviderModels())
}

func (c *Config) applyRawProviderModelMutation(raw map[string]ProviderModelConfig) {
	if c == nil {
		return
	}
	nextState := c.providerModelsStore.nextStateForMutation(raw)
	if len(raw) == 0 {
		c.setProviderModelStoreState(nextState, nil)
		return
	}
	c.setProviderModelStoreState(nextState, raw)
}

// ProviderModelsForSave returns the raw provider_models map that should be serialized on save.
func (c *Config) ProviderModelsForSave() map[string]ProviderModelConfig {
	if c == nil {
		return nil
	}
	return c.providerModelsStore.rawForSave()
}

// ProviderModelsForEdit returns the raw provider_models map that should be shown in editing UIs.
func (c *Config) ProviderModelsForEdit() map[string]ProviderModelConfig {
	if c == nil {
		return nil
	}
	return c.providerModelsStore.rawForEdit(c.effectiveProviderModels())
}
