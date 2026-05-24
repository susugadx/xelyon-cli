package config

import (
	"fmt"
	"strings"
)

func validateReviewIssues(cfg *Config) []ValidationIssue {
	var issues []ValidationIssue
	provider := strings.TrimSpace(cfg.Review.Provider)
	model := strings.TrimSpace(cfg.Review.Model)

	if provider == "" {
		if model == "" {
			return nil
		}
		return []ValidationIssue{{
			Field:      "review.model",
			Value:      model,
			Message:    "review.model は review.provider と一緒に設定してください",
			Suggestion: "review.provider を設定するか、review.model を空にしてください",
			Severity:   ValidationSeverityError,
			CanAutoFix: false,
		}}
	}

	if !isValidProvider(provider) {
		suggested := suggestProvider(provider)
		issues = append(issues, ValidationIssue{
			Field:      "review.provider",
			Value:      provider,
			Message:    fmt.Sprintf("無効な review provider です (有効: %s)", strings.Join(ValidProviders, ", ")),
			Suggestion: suggested,
			Severity:   ValidationSeverityError,
			CanAutoFix: true,
			FixedValue: suggested,
		})
	}

	return issues
}
