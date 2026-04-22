package config

func (c *Config) explicitProviderModelSelection(provider string) (map[string]ProviderModelConfig, string, bool) {
	if c == nil {
		return nil, "", false
	}

	source := c.explicitProviderModelSource()
	key, ok := providerModelLookupKey(source, provider)
	if !ok {
		return source, "", false
	}
	return source, key, true
}

func (c *Config) rawExplicitProviderModelConfig(provider string) (ProviderModelConfig, bool) {
	source, selectedKey, ok := c.explicitProviderModelSelection(provider)
	if !ok {
		return ProviderModelConfig{}, false
	}
	return explicitSelectedProviderModelConfig(source, selectedKey)
}

func (c *Config) selectedProviderModelLookupKey(provider string) (string, bool) {
	if c == nil {
		return "", false
	}
	if _, key, ok := c.explicitProviderModelSelection(provider); ok {
		return key, true
	}
	return providerModelLookupKey(c.effectiveProviderModels(), provider)
}

// GetProviderDefaultModel は provider の実効デフォルトモデルを返す。
// provider_models の明示 override があればそれを使い、無ければ provider built-in default を返す。
func (c *Config) GetProviderDefaultModel(provider string) string {
	if c == nil {
		return ""
	}

	if pm, ok := c.GetProviderModelConfig(provider); ok {
		return pm.DefaultModel
	}

	return ""
}

// GetModelForProvider は GetProviderDefaultModel の後方互換エイリアス。
func (c *Config) GetModelForProvider(provider string) string {
	return c.GetProviderDefaultModel(provider)
}

// GetEffectiveModelForProvider は merged provider config から実行時に使う既定モデルを返す。
// provider_models section の有無にかかわらず provider default を含む実効値を返す。
func (c *Config) GetEffectiveModelForProvider(provider string) string {
	if c == nil {
		return ""
	}
	if pm, ok := c.GetProviderModelConfig(provider); ok {
		return pm.DefaultModel
	}
	return ""
}

// GetSelectedModelForProvider は直接その provider を使うときのモデル解決を返す。
// 優先順位:
// 1. explicit provider_models.<provider>.default_model
// 2. global default_model (default provider のみ。provider と両立すると判断できる場合)
// 3. provider default
func (c *Config) GetSelectedModelForProvider(provider string) string {
	if c == nil {
		return ""
	}

	if pm, ok := c.rawExplicitProviderModelConfig(provider); ok && pm.DefaultModel != "" {
		return pm.DefaultModel
	}
	if SameProviderRuntimeIdentity(provider, c.DefaultProvider) && c.DefaultModel != "" {
		if c.configuredDefaultModelAppliesToProvider(provider, c.DefaultModel) {
			return c.DefaultModel
		}
	}
	if model := c.GetEffectiveModelForProvider(provider); model != "" {
		return model
	}
	return ""
}

// ResolveModelForProvider は GetSelectedModelForProvider の後方互換エイリアス。
func (c *Config) ResolveModelForProvider(provider string) string {
	return c.GetSelectedModelForProvider(provider)
}

// ValidateModelForProvider は任意のモデル名を受け付ける（後方互換のため残す）
// 注: v0.16.0以降、モデル名の検証は行わない
func (c *Config) ValidateModelForProvider(provider, model string) bool {
	_, ok := c.GetProviderModelConfig(provider)
	return ok
}

// GetProviderModelConfig は provider_models を alias fallback 付きで取得する。
// 要求された key を最優先し、存在しない場合のみ sibling alias へ fallback する。
func (c *Config) GetProviderModelConfig(provider string) (ProviderModelConfig, bool) {
	selectedKey, ok := c.selectedProviderModelLookupKey(provider)
	if !ok {
		return ProviderModelConfig{}, false
	}

	override, hasOverride := c.selectedProviderModelConfig(selectedKey)
	return mergeSelectedProviderModelConfig(selectedKey, override, hasOverride), true
}

// GetExplicitProviderModelConfig は明示設定された provider_models entry を返す。
// 要求された key を最優先し、存在しない場合のみ sibling alias へ fallback する。
func (c *Config) GetExplicitProviderModelConfig(provider string) (ProviderModelConfig, bool) {
	source, selectedKey, ok := c.explicitProviderModelSelection(provider)
	if !ok {
		return ProviderModelConfig{}, false
	}

	override, hasOverride := explicitSelectedProviderModelConfig(source, selectedKey)
	return mergeSelectedProviderModelConfig(selectedKey, override, hasOverride), true
}
