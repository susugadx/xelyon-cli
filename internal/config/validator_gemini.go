package config

import "fmt"

func validateGeminiIssues(cfg *Config) []ValidationIssue {
	var issues []ValidationIssue
	if cfg == nil {
		return nil
	}
	if !IsValidGeminiServiceTier(cfg.Gemini.ServiceTier) {
		issues = append(issues, ValidationIssue{
			Field:      "gemini.service_tier",
			Value:      cfg.Gemini.ServiceTier,
			Message:    "Gemini service_tier は standard / flex / priority のいずれかで指定してください",
			Suggestion: fmt.Sprintf("gemini.service_tier を %q に戻すか、%q / %q を指定してください", GeminiServiceTierStandard, GeminiServiceTierFlex, GeminiServiceTierPriority),
			Severity:   ValidationSeverityWarning,
			CanAutoFix: true,
			FixedValue: GeminiServiceTierStandard,
		})
	}
	return append(issues, validateGeminiFunctionCallingIssues(cfg)...)
}
