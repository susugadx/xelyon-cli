package config

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
