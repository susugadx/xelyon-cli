package config

import "strings"

const (
	defaultSkillsRouterUsageRetentionDays = 30
	minSkillsRouterUsageRetentionDays     = 1
	maxSkillsRouterUsageRetentionDays     = 365
)

func defaultSkillsConfig() SkillsConfig {
	return SkillsConfig{
		Router: SkillsRouterConfig{
			Enabled:            true,
			Activation:         SkillsRouterActivationHint,
			UsageLedger:        true,
			UsageRetentionDays: defaultSkillsRouterUsageRetentionDays,
		},
	}
}

// SkillsRouterActivationValues は v1 public config で受け付ける activation 値を返す。
func SkillsRouterActivationValues() []string {
	return []string{
		string(SkillsRouterActivationOff),
		string(SkillsRouterActivationHint),
	}
}

// NormalizeSkillsRouterActivation は Skill Router activation を正規化する。
func NormalizeSkillsRouterActivation(value SkillsRouterActivation) SkillsRouterActivation {
	switch SkillsRouterActivation(strings.ToLower(strings.TrimSpace(string(value)))) {
	case SkillsRouterActivationOff:
		return SkillsRouterActivationOff
	case "", SkillsRouterActivationHint:
		return SkillsRouterActivationHint
	default:
		return value
	}
}

// IsValidSkillsRouterActivation は v1 public config で有効な activation 値か返す。
func IsValidSkillsRouterActivation(value SkillsRouterActivation) bool {
	normalized := NormalizeSkillsRouterActivation(value)
	for _, allowed := range SkillsRouterActivationValues() {
		if string(normalized) == allowed {
			return true
		}
	}
	return false
}

// EffectiveSkillsRouterUsageRetentionDays は usage ledger retention days を既定込みで返す。
func EffectiveSkillsRouterUsageRetentionDays(cfg *Config) int {
	if cfg == nil {
		return defaultSkillsRouterUsageRetentionDays
	}
	days := cfg.Skills.Router.UsageRetentionDays
	if days < minSkillsRouterUsageRetentionDays || days > maxSkillsRouterUsageRetentionDays {
		return defaultSkillsRouterUsageRetentionDays
	}
	return days
}

// SkillsRouterRuntimeHintEnabled は runtime skill hint injection が有効か返す。
func SkillsRouterRuntimeHintEnabled(cfg *Config) bool {
	if cfg == nil {
		return true
	}
	if !cfg.Skills.Router.Enabled {
		return false
	}
	return NormalizeSkillsRouterActivation(cfg.Skills.Router.Activation) == SkillsRouterActivationHint
}
