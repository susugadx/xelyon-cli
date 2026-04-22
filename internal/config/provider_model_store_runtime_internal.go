package config

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
