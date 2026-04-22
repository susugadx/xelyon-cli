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

// ActiveProviderConfigKey は active provider/session の正確な config 側 owner key を返す。
// 明示 provider alias がある場合は default_provider 側の表記に書き換えない。
func ActiveProviderConfigKey(provider string) string {
	return NormalizeProviderName(provider)
}

// FallbackProviderConfigKey は汎用的な config 側 lookup key を返す。
// provider があればそれを優先し、provider が空のときだけ defaultProvider を使う。
func FallbackProviderConfigKey(provider, defaultProvider string) string {
	if normalizedProvider := ActiveProviderConfigKey(provider); normalizedProvider != "" {
		return normalizedProvider
	}
	return NormalizeProviderName(defaultProvider)
}

// PreferredProviderConfigKey は fallback lookup 用の後方互換エイリアス。
// active/session の alias ownership には ActiveProviderConfigKey を使う。
func PreferredProviderConfigKey(provider, defaultProvider string) string {
	return FallbackProviderConfigKey(provider, defaultProvider)
}

func (c *Config) PreferredProviderConfigKey(provider string) string {
	if c == nil {
		return PreferredProviderConfigKey(provider, "")
	}
	return PreferredProviderConfigKey(provider, c.DefaultProvider)
}

// DefaultModelSyncProviderKey は default_model 同期先となる provider config key を返す。
// default_provider が別 runtime identity に編集されていれば、その編集後 provider を優先する。
// それ以外は current session の正確な provider config key を優先し、最後に current default_provider へ fallback する。
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
