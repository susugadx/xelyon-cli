package config

// SyncProviderDefaultModel は provider の default_model override を同期する。
// 値が provider default に戻った場合は明示 override だけを削除する。
func (c *Config) SyncProviderDefaultModel(provider, model string) bool {
	if c == nil {
		return false
	}
	plan := providerDefaultModelSyncPlanFor(provider, model)
	if !plan.valid {
		return false
	}
	if plan.clearExact {
		return c.clearProviderDefaultModelOverrideExactByKey(plan.key)
	}
	c.setProviderDefaultModelOverrideByKey(plan.key, plan.model)
	return true
}

func (c *Config) clearProviderDefaultModelOverrideExact(provider string) bool {
	if c == nil {
		return false
	}

	key, ok := providerDefaultModelConfigKey(provider)
	if !ok {
		return false
	}
	return c.clearProviderDefaultModelOverrideExactByKey(key)
}

func (c *Config) clearProviderDefaultModelOverrideExactByKey(key string) bool {
	if c == nil || key == "" {
		return false
	}
	raw := c.clonedRawProviderModelsForMutation()
	_, ok := raw[key]
	if !ok {
		return true
	}

	clearProviderDefaultModelInRaw(raw, key)
	c.applyRawProviderModelMutation(raw)
	return true
}

func (c *Config) setProviderDefaultModelOverrideByKey(key, model string) {
	if c == nil || key == "" || model == "" {
		return
	}
	raw := c.mutableRawProviderModelsForMutation()
	setProviderDefaultModelInRaw(raw, key, model)
	c.applyRawProviderModelMutation(raw)
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

	clearProviderDefaultModelInRaw(raw, key)
	c.applyRawProviderModelMutation(raw)
	return true
}
