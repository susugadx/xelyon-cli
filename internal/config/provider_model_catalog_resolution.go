package config

import "strings"

// ModelOverrideForProvider は provider/model に対応する model_overrides entry を返す。
func (c *Config) ModelOverrideForProvider(provider, model string) (ModelOverride, bool) {
	if c == nil {
		return ModelOverride{}, false
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return ModelOverride{}, false
	}

	lookupProvider := c.RuntimeProviderConfigKey(provider, model)
	pm, ok := c.GetProviderModelConfig(lookupProvider)
	if !ok || len(pm.ModelOverrides) == 0 {
		return ModelOverride{}, false
	}
	override, ok := pm.ModelOverrides[model]
	return override, ok
}

// ModelCatalogName は token limit / pricing / catalog lookup に使うモデル名を返す。
func (c *Config) ModelCatalogName(provider, model string) string {
	model = strings.TrimSpace(model)
	if model == "" || c == nil {
		return model
	}

	if override, ok := c.ModelOverrideForProvider(provider, model); ok {
		if catalogModel := strings.TrimSpace(override.CatalogModel); catalogModel != "" {
			return catalogModel
		}
	}

	lookupProvider := c.RuntimeProviderConfigKey(provider, model)
	pm, ok := c.GetProviderModelConfig(lookupProvider)
	if ok && pm.DefaultModel == model {
		if catalogModel := strings.TrimSpace(pm.CatalogModel); catalogModel != "" {
			return catalogModel
		}
	}

	return model
}
