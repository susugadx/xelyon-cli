package config

import "strings"

// FindProviderByDefaultModel は実効デフォルトモデルが model と一致する最初の provider を返す。
func (c *Config) FindProviderByDefaultModel(model string) string {
	if c == nil {
		return ""
	}
	for _, provider := range orderedProviderConfigCandidates(c.DefaultProvider) {
		if c.GetProviderDefaultModel(provider) == model {
			return provider
		}
	}
	return ""
}

// FindProviderBySelectedModel は選択モデル解決結果が model と一致する最初の provider を返す。
// 既定 provider の表記を先に確認しつつ、同一 runtime identity の alias も後続で探索する。
func (c *Config) FindProviderBySelectedModel(model string) string {
	if c == nil {
		return ""
	}

	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}

	for _, providerName := range orderedProviderConfigCandidates(c.DefaultProvider) {
		if c.GetSelectedModelForProvider(providerName) == model {
			return providerName
		}
	}

	return ""
}

func (c *Config) selectedModelOwnerWithinRuntimeIdentity(provider, model string) string {
	if c == nil {
		return ""
	}

	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}

	for _, candidate := range ProviderModelLookupKeys(provider) {
		if c.GetSelectedModelForProvider(candidate) == model {
			return candidate
		}
	}

	return ""
}

// RuntimeProviderConfigKey は provider/model の組に対して実行時設定を保持すべき
// config 側 provider key を返す。モデル選択の所有者と周辺設定の所有者を揃えるために使う。
func (c *Config) RuntimeProviderConfigKey(provider, model string) string {
	if c == nil {
		return FallbackProviderConfigKey(provider, "")
	}
	if owner := c.selectedModelOwnerWithinRuntimeIdentity(provider, model); owner != "" {
		return owner
	}
	if owner := c.FindProviderBySelectedModel(model); owner != "" && SameProviderRuntimeIdentity(owner, provider) {
		return owner
	}
	return FallbackProviderConfigKey(provider, c.DefaultProvider)
}

func normalizeModelResolutionInput(currentProvider, model string) (string, string) {
	return CanonicalProviderName(currentProvider), strings.TrimSpace(model)
}

func resolveProviderForModelFallback(currentProvider, model string) string {
	if inferred := InferProviderFromModel(model); inferred != "" {
		return CanonicalProviderName(inferred)
	}
	return currentProvider
}

// ResolveProviderForModel は model を所有すべき実行時 provider を解決する。
// 解決順序は「config の選択モデル」→「provider default」→「モデル名からの推定」。
func (c *Config) ResolveProviderForModel(currentProvider, model string) string {
	currentProvider, model = normalizeModelResolutionInput(currentProvider, model)
	if model == "" {
		return currentProvider
	}

	if providerName := c.FindProviderBySelectedModel(model); providerName != "" {
		return CanonicalProviderName(providerName)
	}
	if providerName := c.FindProviderByDefaultModel(model); providerName != "" {
		return CanonicalProviderName(providerName)
	}
	return resolveProviderForModelFallback(currentProvider, model)
}
