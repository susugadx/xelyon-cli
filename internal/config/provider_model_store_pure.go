package config

import "reflect"

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
	cloned := make(map[string]ProviderModelConfig, len(src))
	for key, value := range src {
		normalized := NormalizeProviderName(key)
		if normalized == "" {
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
		pm.AnthropicVersion == "" &&
		len(pm.AnthropicBeta) == 0 &&
		len(pm.ModelOverrides) == 0
}

func diffProviderModelConfig(base, current ProviderModelConfig) ProviderModelConfig {
	diff := ProviderModelConfig{}

	if current.DefaultModel != "" && current.DefaultModel != base.DefaultModel {
		diff.DefaultModel = current.DefaultModel
	}
	if current.MaxOutputTokens > 0 && current.MaxOutputTokens != base.MaxOutputTokens {
		diff.MaxOutputTokens = current.MaxOutputTokens
	}
	if current.AnthropicVersion != "" && current.AnthropicVersion != base.AnthropicVersion {
		diff.AnthropicVersion = current.AnthropicVersion
	}
	if len(current.AnthropicBeta) > 0 && !reflect.DeepEqual(current.AnthropicBeta, base.AnthropicBeta) {
		diff.AnthropicBeta = append([]string(nil), current.AnthropicBeta...)
	}
	if len(current.ModelOverrides) > 0 && !reflect.DeepEqual(current.ModelOverrides, base.ModelOverrides) {
		diff.ModelOverrides = cloneModelOverrides(current.ModelOverrides)
	}

	return diff
}

func rawProviderModelsFromEffectiveDiff(effective map[string]ProviderModelConfig) map[string]ProviderModelConfig {
	if len(effective) == 0 {
		return nil
	}

	raw := map[string]ProviderModelConfig{}
	for key, pm := range effective {
		normalized := NormalizeProviderName(key)
		if normalized == "" {
			continue
		}

		base, ok := defaultProviderModelConfig(normalized)
		if ok {
			pm = diffProviderModelConfig(base, pm)
		}
		if isZeroProviderModelConfig(pm) {
			continue
		}
		raw[normalized] = cloneProviderModelConfig(pm)
	}

	if len(raw) == 0 {
		return nil
	}
	return raw
}

func buildEffectiveProviderModels(src map[string]ProviderModelConfig) map[string]ProviderModelConfig {
	effective := cloneProviderModelConfigMap(DefaultConfig().ProviderModels)
	if effective == nil {
		effective = map[string]ProviderModelConfig{}
	}

	for key, pm := range src {
		base, ok := defaultProviderModelConfig(key)
		if ok {
			effective[key] = mergeProviderModelConfig(base, pm)
			continue
		}
		effective[key] = cloneProviderModelConfig(pm)
	}

	return effective
}

func normalizeProviderModelsForEdit(providerModels map[string]ProviderModelConfig) map[string]ProviderModelConfig {
	if providerModels == nil {
		return nil
	}
	return cloneProviderModelConfigMap(providerModels)
}
