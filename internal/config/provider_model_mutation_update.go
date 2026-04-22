package config

func providerModelMutationKey(raw map[string]ProviderModelConfig, provider string) (string, bool) {
	return providerModelWriteTargetKey(raw, provider)
}

func currentProviderModelConfigForMutation(raw map[string]ProviderModelConfig, selectedKey string) ProviderModelConfig {
	if raw == nil || selectedKey == "" {
		return ProviderModelConfig{}
	}
	if pm, ok := raw[selectedKey]; ok {
		return cloneProviderModelConfig(pm)
	}
	return ProviderModelConfig{}
}

func (c *Config) mutateProviderModelConfig(provider string, update func(*ProviderModelConfig)) bool {
	if c == nil || update == nil {
		return false
	}

	raw := c.mutableRawProviderModelsForMutation()
	key, ok := providerModelMutationKey(raw, provider)
	if !ok {
		return false
	}

	pm := currentProviderModelConfigForMutation(raw, key)
	update(&pm)
	raw[key] = cloneProviderModelConfig(pm)
	c.applyRawProviderModelMutation(raw)
	return true
}

// SetProviderModelConfig は provider_models の 1 エントリをマージ更新する。
func (c *Config) SetProviderModelConfig(provider string, pm ProviderModelConfig) {
	_ = c.mutateProviderModelConfig(provider, func(existing *ProviderModelConfig) {
		*existing = mergeProviderModelConfig(*existing, pm)
	})
}

// PatchProviderModelConfig は現在のマージ済み値に対するパッチとして
// provider_models の 1 エントリを更新する。
func (c *Config) PatchProviderModelConfig(provider string, patch func(*ProviderModelConfig)) bool {
	return c.mutateProviderModelConfig(provider, patch)
}
