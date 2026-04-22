package config

// SyncProviderDefaultModel は provider の default_model override を同期する。
// 値が provider default に戻った場合は明示 override だけを削除する。
func (c *Config) SyncProviderDefaultModel(provider, model string) bool {
	if c == nil {
		return false
	}
	key := ActiveProviderConfigKey(provider)
	if key == "" {
		return false
	}
	if model == "" {
		return false
	}
	if base, ok := defaultProviderModelConfig(key); ok {
		if model == base.DefaultModel {
			return c.clearProviderDefaultModelOverrideExact(key)
		}
	}

	raw := c.mutableRawProviderModelsForMutation()
	pm := currentProviderModelConfigForMutation(raw, key)
	pm.DefaultModel = model
	raw[key] = cloneProviderModelConfig(pm)
	c.applyRawProviderModelMutation(raw)
	return true
}

func (c *Config) clearProviderDefaultModelOverrideExact(provider string) bool {
	if c == nil {
		return false
	}

	key := ActiveProviderConfigKey(provider)
	if key == "" {
		return false
	}

	raw := c.clonedRawProviderModelsForMutation()
	pm, ok := raw[key]
	if !ok {
		return true
	}

	pm = cloneProviderModelConfig(pm)
	pm.DefaultModel = ""
	if isZeroProviderModelConfig(pm) {
		delete(raw, key)
	} else {
		raw[key] = pm
	}
	c.applyRawProviderModelMutation(raw)
	return true
}

// ClearProviderDefaultModelOverride は provider の default_model override のみ削除する。
// max_output_tokens や model_overrides など他の provider 設定は保持する。
func (c *Config) ClearProviderDefaultModelOverride(provider string) bool {
	if c == nil {
		return false
	}

	raw := c.clonedRawProviderModelsForMutation()
	key, ok := providerModelMutationKey(raw, provider)
	if !ok {
		return true
	}

	pm := currentProviderModelConfigForMutation(raw, key)
	pm.DefaultModel = ""
	if isZeroProviderModelConfig(pm) {
		delete(raw, key)
	} else {
		raw[key] = pm
	}

	c.applyRawProviderModelMutation(raw)
	return true
}
