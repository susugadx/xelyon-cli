package config

func defaultProviderModelConfig(provider string) (ProviderModelConfig, bool) {
	keys := ProviderModelLookupKeys(provider)
	if len(keys) == 0 {
		return ProviderModelConfig{}, false
	}

	defaults := DefaultConfig()
	merged := ProviderModelConfig{}
	found := false
	for i := len(keys) - 1; i >= 0; i-- {
		key := keys[i]
		if pm, ok := defaults.ProviderModels[key]; ok {
			merged = mergeProviderModelConfig(merged, pm)
			found = true
		}
	}
	return merged, found
}

func providerModelBaseConfig(selectedKey string) ProviderModelConfig {
	if pm, ok := defaultProviderModelConfig(selectedKey); ok {
		return pm
	}
	return ProviderModelConfig{}
}

func mergeSelectedProviderModelConfig(selectedKey string, override ProviderModelConfig, hasOverride bool) ProviderModelConfig {
	base := providerModelBaseConfig(selectedKey)
	if !hasOverride {
		return base
	}
	return mergeProviderModelConfig(base, override)
}

func explicitSelectedProviderModelConfig(source map[string]ProviderModelConfig, selectedKey string) (ProviderModelConfig, bool) {
	if source == nil || selectedKey == "" {
		return ProviderModelConfig{}, false
	}
	pm, ok := source[selectedKey]
	if !ok {
		return ProviderModelConfig{}, false
	}
	return cloneProviderModelConfig(pm), true
}

func (c *Config) selectedProviderModelConfig(selectedKey string) (ProviderModelConfig, bool) {
	if c == nil || selectedKey == "" {
		return ProviderModelConfig{}, false
	}
	if pm, ok := c.providerModelsStore.rawConfig(selectedKey); ok {
		return pm, true
	}
	if pm, ok := c.effectiveProviderModelConfig(selectedKey); ok {
		return pm, true
	}
	return ProviderModelConfig{}, false
}
