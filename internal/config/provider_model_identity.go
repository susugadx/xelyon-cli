package config

import "strings"

// NormalizeProviderName は provider 名を設定 lookup 用に正規化する。
func NormalizeProviderName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// CanonicalProviderName は provider 名を実行時 canonical 名に変換する。
func CanonicalProviderName(name string) string {
	switch NormalizeProviderName(name) {
	case "anthropic":
		return "claude"
	default:
		return NormalizeProviderName(name)
	}
}

// SameProviderRuntimeIdentity は alias/canonical を吸収して同一 runtime provider か判定する。
func SameProviderRuntimeIdentity(a, b string) bool {
	canonicalA := CanonicalProviderName(a)
	canonicalB := CanonicalProviderName(b)
	return canonicalA != "" && canonicalA == canonicalB
}

// ActiveProviderConfigKey returns the exact config-side owner key for an active provider/session.
// When an explicit provider alias is known, it must not be rewritten to default_provider spelling.
func ActiveProviderConfigKey(provider string) string {
	return NormalizeProviderName(provider)
}

// FallbackProviderConfigKey returns a generic config-side lookup key.
// The explicit provider wins when present; defaultProvider is used only when provider is empty.
func FallbackProviderConfigKey(provider, defaultProvider string) string {
	if normalizedProvider := ActiveProviderConfigKey(provider); normalizedProvider != "" {
		return normalizedProvider
	}
	return NormalizeProviderName(defaultProvider)
}

// PreferredProviderConfigKey is kept as a compatibility alias for fallback lookup.
// Active/session alias ownership must use ActiveProviderConfigKey instead.
func PreferredProviderConfigKey(provider, defaultProvider string) string {
	return FallbackProviderConfigKey(provider, defaultProvider)
}

func (c *Config) PreferredProviderConfigKey(provider string) string {
	if c == nil {
		return PreferredProviderConfigKey(provider, "")
	}
	return PreferredProviderConfigKey(provider, c.DefaultProvider)
}

// DefaultModelSyncProviderKey returns the provider config key that should receive a default_model sync.
// If default_provider was edited to a different runtime identity, the edited provider wins.
// Otherwise the current session's exact provider config key wins, falling back to current default_provider.
func DefaultModelSyncProviderKey(currentSessionProviderConfigKey, currentDefaultProvider, initialDefaultProvider string) string {
	normalizedCurrentDefault := NormalizeProviderName(currentDefaultProvider)
	normalizedInitialDefault := NormalizeProviderName(initialDefaultProvider)
	if normalizedCurrentDefault != normalizedInitialDefault && !SameProviderRuntimeIdentity(normalizedCurrentDefault, normalizedInitialDefault) {
		return normalizedCurrentDefault
	}
	if normalizedSessionProvider := ActiveProviderConfigKey(currentSessionProviderConfigKey); normalizedSessionProvider != "" {
		return normalizedSessionProvider
	}
	return FallbackProviderConfigKey("", currentDefaultProvider)
}

func (c *Config) DefaultModelSyncProviderKey(currentSessionProviderConfigKey, initialDefaultProvider string) string {
	if c == nil {
		return DefaultModelSyncProviderKey(currentSessionProviderConfigKey, "", initialDefaultProvider)
	}
	return DefaultModelSyncProviderKey(currentSessionProviderConfigKey, c.DefaultProvider, initialDefaultProvider)
}

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

func providerModelLookupKey(src map[string]ProviderModelConfig, provider string) (string, bool) {
	keys := ProviderModelLookupKeys(provider)
	if len(keys) == 0 || src == nil {
		return "", false
	}

	for _, key := range keys {
		if _, ok := src[key]; ok {
			return key, true
		}
	}

	return "", false
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
	if key, ok := providerModelLookupKey(raw, provider); ok {
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
	if key, ok := providerModelLookupKey(raw, provider); ok {
		return []string{key}
	}
	return nil
}
