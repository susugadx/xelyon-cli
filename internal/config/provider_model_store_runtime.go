package config

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

func (c *Config) explicitProviderModelSource() map[string]ProviderModelConfig {
	if c == nil {
		return nil
	}
	return c.providerModelsStore.explicitSource(c.effectiveProviderModels())
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

func (c *Config) applyProviderModelEditTransition(transition providerModelStoreEditTransition) {
	if c == nil {
		return
	}
	if transition.resetInMemoryEffective {
		c.resetInMemoryEffectiveProviderModels()
		return
	}
	c.setProviderModelStoreState(transition.state, transition.raw)
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
	transition := c.providerModelsStore.transitionAfterRawMutation(raw)
	c.setProviderModelStoreState(transition.state, transition.raw)
}

// ResetProviderModelsForEdit は provider_models を既定の「section absent」状態へ戻す。
func (c *Config) ResetProviderModelsForEdit() {
	if c == nil {
		return
	}
	c.setProviderModelStoreState(providerModelSectionStateAbsent, nil)
}

// SetProviderModelsForEdit は編集UI向け provider_models バッキングマップを更新する。
func (c *Config) SetProviderModelsForEdit(providerModels map[string]ProviderModelConfig) {
	if c == nil {
		return
	}
	if providerModels == nil {
		c.ResetProviderModelsForEdit()
		return
	}

	cloned := normalizeProviderModelsForEdit(providerModels)
	transition := c.providerModelsStore.transitionAfterEditingEntries(cloned)
	c.applyProviderModelEditTransition(transition)
}

// ProviderModelsForSave は保存時にシリアライズすべき raw provider_models マップを返す。
func (c *Config) ProviderModelsForSave() map[string]ProviderModelConfig {
	if c == nil {
		return nil
	}
	return c.providerModelsStore.rawForSave()
}

// ProviderModelsForEdit は編集UIに表示すべき raw provider_models マップを返す。
func (c *Config) ProviderModelsForEdit() map[string]ProviderModelConfig {
	if c == nil {
		return nil
	}
	return c.providerModelsStore.rawForEdit(c.effectiveProviderModels())
}
