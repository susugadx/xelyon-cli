package config

import "fmt"

func validateSkillsIssues(cfg *Config) []ValidationIssue {
	if cfg == nil {
		return nil
	}
	var issues []ValidationIssue
	if !IsValidSkillsRouterActivation(cfg.Skills.Router.Activation) {
		issues = append(issues, ValidationIssue{
			Field:      "skills.router.activation",
			Value:      string(cfg.Skills.Router.Activation),
			Message:    "Skill Router activation は off または hint を指定してください。auto は v1 public config では未対応です",
			Suggestion: string(SkillsRouterActivationHint),
			Severity:   ValidationSeverityError,
			CanAutoFix: true,
			FixedValue: string(SkillsRouterActivationHint),
		})
	}
	days := cfg.Skills.Router.UsageRetentionDays
	if days < minSkillsRouterUsageRetentionDays || days > maxSkillsRouterUsageRetentionDays {
		issues = append(issues, ValidationIssue{
			Field:      "skills.router.usage_retention_days",
			Value:      fmt.Sprintf("%d", days),
			Message:    fmt.Sprintf("usage_retention_days は %d-%d の範囲で指定してください。無効化は usage_ledger: false を使ってください", minSkillsRouterUsageRetentionDays, maxSkillsRouterUsageRetentionDays),
			Suggestion: fmt.Sprintf("%d", defaultSkillsRouterUsageRetentionDays),
			Severity:   ValidationSeverityError,
			CanAutoFix: true,
			FixedValue: defaultSkillsRouterUsageRetentionDays,
		})
	}
	return issues
}
