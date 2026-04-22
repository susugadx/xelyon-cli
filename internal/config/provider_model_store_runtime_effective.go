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
