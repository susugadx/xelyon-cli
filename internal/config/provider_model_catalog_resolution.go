package config

import "strings"

// ModelCatalogResolution は provider/model から catalog lookup 用モデル名への解決結果です。
type ModelCatalogResolution struct {
	Model                    string
	ConfiguredWithoutCatalog bool
}

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
	return c.ResolveModelCatalog(provider, model).Model
}

// ResolveModelCatalog は token limit / pricing / catalog lookup に使うモデル名と、
// その名前が catalog_model なしの設定値かどうかを返す。
func (c *Config) ResolveModelCatalog(provider, model string) ModelCatalogResolution {
	model = strings.TrimSpace(model)
	if model == "" || c == nil {
		return ModelCatalogResolution{Model: model}
	}

	lookupProvider := c.RuntimeProviderConfigKey(provider, model)
	pm, hasProviderModelConfig := c.rawExplicitProviderModelConfig(lookupProvider)

	if override, ok := c.ModelOverrideForProvider(provider, model); ok {
		if catalogModel := strings.TrimSpace(override.CatalogModel); catalogModel != "" {
			return ModelCatalogResolution{Model: catalogModel}
		}
		if hasProviderModelConfig && strings.TrimSpace(pm.DefaultModel) == model {
			if catalogModel := strings.TrimSpace(pm.CatalogModel); catalogModel != "" {
				return ModelCatalogResolution{Model: catalogModel}
			}
		}
		return ModelCatalogResolution{
			Model:                    model,
			ConfiguredWithoutCatalog: true,
		}
	}

	if hasProviderModelConfig && strings.TrimSpace(pm.DefaultModel) == model {
		if catalogModel := strings.TrimSpace(pm.CatalogModel); catalogModel != "" {
			return ModelCatalogResolution{Model: catalogModel}
		}
		return ModelCatalogResolution{
			Model:                    model,
			ConfiguredWithoutCatalog: true,
		}
	}

	if strings.TrimSpace(c.DefaultModel) == model &&
		SameProviderRuntimeIdentity(provider, c.DefaultProvider) &&
		c.configuredDefaultModelAppliesToProvider(provider, model) {
		return ModelCatalogResolution{
			Model:                    model,
			ConfiguredWithoutCatalog: true,
		}
	}

	return ModelCatalogResolution{Model: model}
}
