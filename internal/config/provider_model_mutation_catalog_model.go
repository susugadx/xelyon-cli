package config

// ClearProviderCatalogModel は provider_models の provider-level catalog_model だけを削除する。
// default_model、max_output_tokens、model_overrides など他の設定は保持する。
func (c *Config) ClearProviderCatalogModel(provider string) bool {
	if c == nil {
		return false
	}

	raw := c.clonedRawProviderModelsForMutation()
	targets := providerModelDeleteTargetKeys(raw, provider)
	if len(targets) == 0 {
		return true
	}

	for _, key := range targets {
		clearProviderCatalogModelInRaw(raw, key)
	}
	c.applyRawProviderModelMutation(raw)
	return true
}

func clearProviderCatalogModelInRaw(raw map[string]ProviderModelConfig, key string) {
	if raw == nil || key == "" {
		return
	}
	pm := currentProviderModelConfigForMutation(raw, key)
	pm.CatalogModel = ""
	if isZeroProviderModelConfig(pm) {
		delete(raw, key)
		return
	}
	raw[key] = cloneProviderModelConfig(pm)
}
