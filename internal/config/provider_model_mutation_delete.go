package config

type providerModelDeleteAction struct {
	requestedKey string
	deleteKeys   []string
}

func providerModelDeleteActionFor(raw map[string]ProviderModelConfig, provider string) (providerModelDeleteAction, bool) {
	requestedKey, ok := providerModelRequestedKey(provider)
	if !ok {
		return providerModelDeleteAction{}, false
	}
	return providerModelDeleteAction{
		requestedKey: requestedKey,
		deleteKeys:   providerModelDeleteTargetKeys(raw, provider),
	}, true
}

func deleteProviderModelKeys(raw map[string]ProviderModelConfig, keys []string) {
	if raw == nil {
		return
	}
	for _, key := range keys {
		delete(raw, key)
	}
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

func (c *Config) applyProviderModelDeleteAction(action providerModelDeleteAction) {
	if c == nil {
		return
	}
	if len(action.deleteKeys) > 0 {
		c.deleteProviderModelOverridesByKeys(action.deleteKeys)
		return
	}
	c.applyProviderModelDeleteFallback(action.requestedKey)
}

func (c *Config) deleteProviderModelOverridesByKeys(deleteKeys []string) {
	if c == nil || len(deleteKeys) == 0 {
		return
	}
	raw := c.clonedRawProviderModelsForMutation()
	deleteProviderModelKeys(raw, deleteKeys)
	c.applyRawProviderModelMutation(raw)
}

// DeleteProviderModelConfig は provider_models の指定 provider エントリを削除する。
func (c *Config) DeleteProviderModelConfig(provider string) {
	if c == nil {
		return
	}
	raw := c.clonedRawProviderModelsForMutation()
	action, ok := providerModelDeleteActionFor(raw, provider)
	if !ok {
		return
	}
	c.applyProviderModelDeleteAction(action)
}
