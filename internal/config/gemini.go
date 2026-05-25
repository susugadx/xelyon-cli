package config

import "strings"

const (
	// GeminiServiceTierStandard は Gemini の既定 service_tier。
	GeminiServiceTierStandard = "standard"
	// GeminiServiceTierFlex は latency よりコストを優先する Gemini Flex tier。
	GeminiServiceTierFlex = "flex"
	// GeminiServiceTierPriority は低遅延・高信頼性を優先する Gemini Priority tier。
	GeminiServiceTierPriority = "priority"
)

// GeminiServiceTierValues は config で選択できる Gemini service_tier。
func GeminiServiceTierValues() []string {
	return []string{GeminiServiceTierStandard, GeminiServiceTierFlex, GeminiServiceTierPriority}
}

// NormalizeGeminiServiceTier は空値や大小文字差を含む service_tier を正規化する。
func NormalizeGeminiServiceTier(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", GeminiServiceTierStandard:
		return GeminiServiceTierStandard
	case GeminiServiceTierFlex:
		return GeminiServiceTierFlex
	case GeminiServiceTierPriority:
		return GeminiServiceTierPriority
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

// IsValidGeminiServiceTier は service_tier が既知値か返す。
func IsValidGeminiServiceTier(value string) bool {
	normalized := NormalizeGeminiServiceTier(value)
	for _, allowed := range GeminiServiceTierValues() {
		if normalized == allowed {
			return true
		}
	}
	return false
}

// GeminiServiceTier は config の Gemini service_tier を既定込みで返す。
func (c *Config) GeminiServiceTier() string {
	if c == nil {
		return GeminiServiceTierStandard
	}
	serviceTier := NormalizeGeminiServiceTier(c.Gemini.ServiceTier)
	if !IsValidGeminiServiceTier(serviceTier) {
		return GeminiServiceTierStandard
	}
	return serviceTier
}

// GeminiRequestServiceTier は Gemini request body に明示する service_tier を返す。
// standard は API 既定値として扱うため、request には出さない。
func GeminiRequestServiceTier(c *Config) string {
	serviceTier := c.GeminiServiceTier()
	if serviceTier == GeminiServiceTierStandard {
		return ""
	}
	return serviceTier
}
