package config

func (c *Config) explicitProviderModelSelection(provider string) (map[string]ProviderModelConfig, string, bool) {
	if c == nil {
		return nil, "", false
	}

	source := c.explicitProviderModelSource()
	key, ok := providerModelLookupKey(source, provider)
	if !ok {
		return source, "", false
	}
	return source, key, true
}

func (c *Config) rawExplicitProviderModelConfig(provider string) (ProviderModelConfig, bool) {
	source, selectedKey, ok := c.explicitProviderModelSelection(provider)
	if !ok {
		return ProviderModelConfig{}, false
	}
	return explicitSelectedProviderModelConfig(source, selectedKey)
}

func (c *Config) selectedProviderModelLookupKey(provider string) (string, bool) {
	if c == nil {
		return "", false
	}
	if _, key, ok := c.explicitProviderModelSelection(provider); ok {
		return key, true
	}
	return providerModelLookupKey(c.effectiveProviderModels(), provider)
}
