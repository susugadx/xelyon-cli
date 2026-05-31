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
		if model != "" {
			issues = append(issues, ValidationIssue{
				Field:      "review.model",
				Value:      model,
				Message:    "review.model は review.provider と一緒に設定してください",
				Suggestion: "review.provider を設定するか、review.model を空にしてください",
				Severity:   ValidationSeverityError,
				CanAutoFix: false,
			})
		}
	} else if !isValidProvider(provider) {
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

	issues = append(issues, validateReviewWebSearchEvidenceIssues(cfg)...)

	return issues
}

func validateReviewWebSearchEvidenceIssues(cfg *Config) []ValidationIssue {
	if cfg == nil || !cfg.Review.WebSearchEvidence.Enabled {
		return nil
	}

	webSearchEvidence := cfg.Review.WebSearchEvidence
	var issues []ValidationIssue
	if webSearchEvidence.MaxQueries <= 0 {
		issues = append(issues, ValidationIssue{
			Field:      "review.web_search_evidence.max_queries",
			Value:      fmt.Sprintf("%d", webSearchEvidence.MaxQueries),
			Message:    "web_search_evidence.enabled=true の場合は max_queries に正の整数を指定してください",
			Suggestion: "3",
			Severity:   ValidationSeverityError,
			CanAutoFix: false,
		})
	}
	if webSearchEvidence.MaxResultsPerQuery <= 0 {
		issues = append(issues, ValidationIssue{
			Field:      "review.web_search_evidence.max_results_per_query",
			Value:      fmt.Sprintf("%d", webSearchEvidence.MaxResultsPerQuery),
			Message:    "web_search_evidence.enabled=true の場合は max_results_per_query に正の整数を指定してください",
			Suggestion: "3",
			Severity:   ValidationSeverityError,
			CanAutoFix: false,
		})
	}
	return issues
}
