package config

import "strings"

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
// 解決順序は「current provider の選択モデル」→「config の選択モデル」→「provider default」→「モデル名からの推定」。
func (c *Config) ResolveProviderForModel(currentProvider, model string) string {
	currentProvider, model = normalizeModelResolutionInput(currentProvider, model)
	if model == "" {
		return currentProvider
	}

	if owner := c.selectedModelOwnerWithinRuntimeIdentity(currentProvider, model); owner != "" {
		return CanonicalProviderName(owner)
	}
	if providerName := c.FindProviderBySelectedModel(model); providerName != "" {
		return CanonicalProviderName(providerName)
	}
	if providerName := c.FindProviderByDefaultModel(model); providerName != "" {
		return CanonicalProviderName(providerName)
	}
	return resolveProviderForModelFallback(currentProvider, model)
}
