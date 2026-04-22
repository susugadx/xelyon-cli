package config

// ProviderModelLookupKeys は provider_models lookup の優先キーを返す。
// 先頭は常に入力値に最も近いキー、後続は互換 alias/canonical fallback。
func ProviderModelLookupKeys(provider string) []string {
	normalized := NormalizeProviderName(provider)
	if normalized == "" {
		return nil
	}

	keys := []string{normalized}
	switch normalized {
	case "anthropic":
		keys = append(keys, "claude")
	case "claude":
		keys = append(keys, "anthropic")
	}
	return keys
}

func providerModelRequestedKey(provider string) (string, bool) {
	key := ActiveProviderConfigKey(provider)
	if key == "" {
		return "", false
	}
	return key, true
}

func providerModelLookupFallbackKeys(provider string) []string {
	keys := ProviderModelLookupKeys(provider)
	if len(keys) <= 1 {
		return nil
	}
	return keys[1:]
}

func providerModelFirstExistingKey(src map[string]ProviderModelConfig, keys []string) (string, bool) {
	if src == nil || len(keys) == 0 {
		return "", false
	}
	for _, key := range keys {
		if _, ok := src[key]; ok {
			return key, true
		}
	}
	return "", false
}

func providerModelLookupKey(src map[string]ProviderModelConfig, provider string) (string, bool) {
	return providerModelFirstExistingKey(src, ProviderModelLookupKeys(provider))
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
