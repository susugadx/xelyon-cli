package config

import "github.com/susugadx/xelyon-cli/internal/llmcatalog"

func defaultProviderModels() map[string]ProviderModelConfig {
	defaults := llmcatalog.DefaultProviderModelDescriptors()
	out := make(map[string]ProviderModelConfig, len(defaults))
	for key, model := range defaults {
		out[key] = ProviderModelConfig{
			DefaultModel:     model.DefaultModel,
			MaxOutputTokens:  model.MaxOutputTokens,
			AnthropicVersion: model.AnthropicVersion,
			AnthropicBeta:    model.AnthropicBeta,
		}
	}
	return out
}

func defaultProviderModelStore() providerModelStore {
	return providerModelStore{
		state: providerModelSectionStateInMemoryEffectiveOnly,
	}
}
