package config

func deleteProviderModelKeys(raw map[string]ProviderModelConfig, keys []string) {
	if raw == nil {
		return
	}
	for _, key := range keys {
		delete(raw, key)
	}
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

func (c *Config) normalizeEditableProviderModels(providerModels map[string]ProviderModelConfig) map[string]ProviderModelConfig {
	if providerModels == nil {
		return nil
	}
	return cloneProviderModelConfigMap(providerModels)
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

// PatchProviderModelConfig updates one provider_models entry as a patch against the merged current value.
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
	raw := c.clonedRawProviderModelsForMutation()
	if deleteKeys := providerModelDeleteTargetKeys(raw, provider); len(deleteKeys) > 0 {
		deleteProviderModelKeys(raw, deleteKeys)
		c.applyRawProviderModelMutation(raw)
		return
	}
	if base, ok := defaultProviderModelConfig(key); ok {
		c.setEffectiveProviderModelConfig(key, base)
		return
	}
	c.deleteEffectiveProviderModelConfig(key)
}

func (c *Config) selectedProviderModelWriteKey(provider string) (string, bool) {
	if c == nil {
		return "", false
	}
	return providerModelWriteTargetKey(c.explicitProviderModelSource(), provider)
}

// ProviderModelWriteKey は provider_models の更新先キーを返す。
func (c *Config) ProviderModelWriteKey(provider string) (string, bool) {
	return c.selectedProviderModelWriteKey(provider)
}

// UpdateExistingProviderModelConfig は provider_models エントリを更新する。
// raw provider_models が未定義でも、既知 provider なら保存対象の entry を新規作成する。
func (c *Config) UpdateExistingProviderModelConfig(provider string, update func(*ProviderModelConfig)) bool {
	return c.PatchProviderModelConfig(provider, update)
}

// SyncProviderDefaultModel syncs the default_model override for a provider.
// If the value returns to the provider default, only the explicit override is removed.
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

// ResetProviderModelsForEdit restores provider_models to the default "section absent" state.
func (c *Config) ResetProviderModelsForEdit() {
	if c == nil {
		return
	}
	c.setProviderModelStoreState(providerModelSectionStateAbsent, nil)
}

// SetProviderModelsForEdit updates the editable provider_models backing map.
func (c *Config) SetProviderModelsForEdit(providerModels map[string]ProviderModelConfig) {
	if c == nil {
		return
	}
	if providerModels == nil {
		c.ResetProviderModelsForEdit()
		return
	}

	cloned := c.normalizeEditableProviderModels(providerModels)
	transition := c.providerModelsStore.transitionAfterEditingEntries(cloned)
	if transition.resetInMemoryEffective {
		c.resetInMemoryEffectiveProviderModels()
		return
	}
	c.setProviderModelStoreState(transition.state, transition.raw)
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

// ClearProviderDefaultModelOverride removes only the default_model override for a provider.
// Other provider settings such as max_output_tokens and model_overrides are preserved.
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
