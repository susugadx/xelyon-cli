package config

import "reflect"

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
