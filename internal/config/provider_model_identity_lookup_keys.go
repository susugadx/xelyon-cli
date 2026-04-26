package config

import "github.com/susugadx/xelyon-cli/internal/llmcatalog"

// ProviderModelLookupKeys は provider_models lookup の優先キーを返す。
// 先頭は常に入力値に最も近いキー、後続は互換 alias/canonical fallback。
func ProviderModelLookupKeys(provider string) []string {
	return llmcatalog.ProviderModelLookupKeys(provider)
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
