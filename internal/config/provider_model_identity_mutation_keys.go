package config

func providerModelRequestedKey(provider string) (string, bool) {
	key := ActiveProviderConfigKey(provider)
	if key == "" {
		return "", false
	}
	return key, true
}

func providerModelWriteTargetKey(raw map[string]ProviderModelConfig, provider string) (string, bool) {
	requestedKey, ok := providerModelRequestedKey(provider)
	if !ok {
		return "", false
	}
	if raw == nil {
		return requestedKey, true
	}
	if _, ok := raw[requestedKey]; ok {
		return requestedKey, true
	}
	if key, ok := providerModelFirstExistingKey(raw, providerModelLookupFallbackKeys(provider)); ok {
		return key, true
	}
	return requestedKey, true
}

func providerModelDeleteTargetKeys(raw map[string]ProviderModelConfig, provider string) []string {
	requestedKey, ok := providerModelRequestedKey(provider)
	if !ok || raw == nil {
		return nil
	}
	if _, ok := raw[requestedKey]; ok {
		return []string{requestedKey}
	}
	if key, ok := providerModelFirstExistingKey(raw, providerModelLookupFallbackKeys(provider)); ok {
		return []string{key}
	}
	return nil
}
