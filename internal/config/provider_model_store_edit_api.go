package config

// ResetProviderModelsForEdit は provider_models を既定の「section absent」状態へ戻す。
func (c *Config) ResetProviderModelsForEdit() {
	if c == nil {
		return
	}
	c.setProviderModelStoreState(providerModelSectionStateAbsent, nil)
}

// SetProviderModelsForEdit は編集UI向け provider_models バッキングマップを更新する。
func (c *Config) SetProviderModelsForEdit(providerModels map[string]ProviderModelConfig) {
	if c == nil {
		return
	}
	if providerModels == nil {
		c.ResetProviderModelsForEdit()
		return
	}

	cloned := normalizeProviderModelsForEdit(providerModels)
	transition := c.providerModelsStore.transitionAfterEditingEntries(cloned)
	c.applyProviderModelEditTransition(transition)
}

// ProviderModelsForSave は保存時にシリアライズすべき raw provider_models マップを返す。
func (c *Config) ProviderModelsForSave() map[string]ProviderModelConfig {
	if c == nil {
		return nil
	}
	return c.providerModelsStore.rawForSave()
}

// ProviderModelsForEdit は編集UIに表示すべき raw provider_models マップを返す。
func (c *Config) ProviderModelsForEdit() map[string]ProviderModelConfig {
	if c == nil {
		return nil
	}
	return c.providerModelsStore.rawForEdit(c.effectiveProviderModels())
}
