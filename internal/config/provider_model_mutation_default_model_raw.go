package config

func setProviderDefaultModelInRaw(raw map[string]ProviderModelConfig, key, model string) {
	if raw == nil || key == "" || model == "" {
		return
	}
	pm := currentProviderModelConfigForMutation(raw, key)
	pm.DefaultModel = model
	raw[key] = cloneProviderModelConfig(pm)
}

func clearProviderDefaultModelInRaw(raw map[string]ProviderModelConfig, key string) {
	if raw == nil || key == "" {
		return
	}
	pm := currentProviderModelConfigForMutation(raw, key)
	pm.DefaultModel = ""
	if isZeroProviderModelConfig(pm) {
		delete(raw, key)
		return
	}
	raw[key] = cloneProviderModelConfig(pm)
}
