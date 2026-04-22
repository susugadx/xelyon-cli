package config

func deleteProviderModelKeys(raw map[string]ProviderModelConfig, keys []string) {
	if raw == nil {
		return
	}
	for _, key := range keys {
		delete(raw, key)
	}
}

func (c *Config) deleteProviderModelOverrides(provider string) bool {
	if c == nil {
		return false
	}

	raw := c.clonedRawProviderModelsForMutation()
	deleteKeys := providerModelDeleteTargetKeys(raw, provider)
	if len(deleteKeys) == 0 {
		return false
	}

	deleteProviderModelKeys(raw, deleteKeys)
	c.applyRawProviderModelMutation(raw)
	return true
}

func (c *Config) applyProviderModelDeleteFallback(key string) {
	if c == nil || key == "" {
		return
	}

	if base, ok := defaultProviderModelConfig(key); ok {
		c.setEffectiveProviderModelConfig(key, base)
		return
	}
	c.deleteEffectiveProviderModelConfig(key)
}

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

func (c *Config) DeleteProviderModelConfig(provider string) {
	if c == nil {
		return
	}

	key, ok := providerModelRequestedKey(provider)
	if !ok {
		return
	}

	if c.deleteProviderModelOverrides(provider) {
		return
	}

	c.applyProviderModelDeleteFallback(key)
}

// ProviderModelWriteKey は provider_models の更新先キーを返す。
func (c *Config) ProviderModelWriteKey(provider string) (string, bool) {
	if c == nil {
		return "", false
	}
	return providerModelWriteTargetKey(c.explicitProviderModelSource(), provider)
}

// UpdateExistingProviderModelConfig は provider_models エントリを更新する。
// raw provider_models が未定義でも、既知 provider なら保存対象の entry を新規作成する。
func (c *Config) UpdateExistingProviderModelConfig(provider string, update func(*ProviderModelConfig)) bool {
	return c.PatchProviderModelConfig(provider, update)
}
