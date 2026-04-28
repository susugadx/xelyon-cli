package config

import (
	"sort"
	"strings"
)

func cloneModelOverrides(src map[string]ModelOverride) map[string]ModelOverride {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string]ModelOverride, len(src))
	for key, value := range src {
		cloned[key] = value
	}
	return cloned
}

func cloneProviderModelConfig(src ProviderModelConfig) ProviderModelConfig {
	cloned := src
	cloned.AnthropicBeta = append([]string(nil), src.AnthropicBeta...)
	cloned.ModelOverrides = cloneModelOverrides(src.ModelOverrides)
	return cloned
}

func cloneProviderModelConfigMap(src map[string]ProviderModelConfig) map[string]ProviderModelConfig {
	if src == nil {
		return nil
	}
	keys := make([]string, 0, len(src))
	for key := range src {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left := keys[i]
		right := keys[j]
		leftOwner := ActiveProviderConfigKey(left)
		rightOwner := ActiveProviderConfigKey(right)
		if leftOwner != rightOwner {
			if leftOwner == "" {
				return false
			}
			if rightOwner == "" {
				return true
			}
			return leftOwner < rightOwner
		}

		// 同一 owner へ正規化されるキーは、display-name/alias を先に適用し、
		// owner key（例: azure）を後で適用して上書き優先を安定化する。
		leftOwnerKey := strings.TrimSpace(strings.ToLower(left)) == leftOwner
		rightOwnerKey := strings.TrimSpace(strings.ToLower(right)) == rightOwner
		if leftOwnerKey != rightOwnerKey {
			return !leftOwnerKey && rightOwnerKey
		}
		return left < right
	})

	cloned := make(map[string]ProviderModelConfig, len(src))
	for _, key := range keys {
		value := src[key]
		normalized := ActiveProviderConfigKey(key)
		if normalized == "" {
			continue
		}
		if current, exists := cloned[normalized]; exists {
			cloned[normalized] = mergeProviderModelConfig(current, value)
			continue
		}
		cloned[normalized] = cloneProviderModelConfig(value)
	}
	return cloned
}

func mergeProviderModelConfig(base, override ProviderModelConfig) ProviderModelConfig {
	merged := base

	if override.DefaultModel != "" {
		merged.DefaultModel = override.DefaultModel
	}
	if override.MaxOutputTokens > 0 {
		merged.MaxOutputTokens = override.MaxOutputTokens
	}
	if override.CatalogModel != "" {
		merged.CatalogModel = override.CatalogModel
	}
	if override.AnthropicVersion != "" {
		merged.AnthropicVersion = override.AnthropicVersion
	}
	if len(override.AnthropicBeta) > 0 {
		merged.AnthropicBeta = append([]string(nil), override.AnthropicBeta...)
	}

	if len(base.ModelOverrides) == 0 && len(override.ModelOverrides) == 0 {
		merged.ModelOverrides = nil
		return merged
	}

	merged.ModelOverrides = cloneModelOverrides(base.ModelOverrides)
	if merged.ModelOverrides == nil {
		merged.ModelOverrides = map[string]ModelOverride{}
	}
	for key, value := range override.ModelOverrides {
		merged.ModelOverrides[key] = value
	}

	return merged
}

func isZeroProviderModelConfig(pm ProviderModelConfig) bool {
	return pm.DefaultModel == "" &&
		pm.MaxOutputTokens == 0 &&
		pm.CatalogModel == "" &&
		pm.AnthropicVersion == "" &&
		len(pm.AnthropicBeta) == 0 &&
		len(pm.ModelOverrides) == 0
}

func normalizeProviderModelsForEdit(providerModels map[string]ProviderModelConfig) map[string]ProviderModelConfig {
	if providerModels == nil {
		return nil
	}
	return cloneProviderModelConfigMap(providerModels)
}
