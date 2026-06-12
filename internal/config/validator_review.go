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

	issues = append(issues, validateReviewThinkingIssues(cfg)...)
	issues = append(issues, validateReviewWebSearchEvidenceIssues(cfg)...)

	return issues
}

func validateReviewThinkingIssues(cfg *Config) []ValidationIssue {
	if cfg == nil {
		return nil
	}

	var issues []ValidationIssue
	mode := NormalizeReviewThinkingMode(cfg.Review.Thinking.Mode)
	switch mode {
	case ReviewThinkingModeInherit, ReviewThinkingModeOff, ReviewThinkingModeOn:
	default:
		issues = append(issues, ValidationIssue{
			Field:      "review.thinking.mode",
			Value:      strings.TrimSpace(string(cfg.Review.Thinking.Mode)),
			Message:    "review.thinking.mode は inherit/off/on のいずれかを指定してください",
			Suggestion: string(ReviewThinkingModeInherit),
			Severity:   ValidationSeverityError,
			CanAutoFix: true,
			FixedValue: string(ReviewThinkingModeInherit),
		})
	}

	level := strings.TrimSpace(cfg.Review.Thinking.Level)
	if !IsValidReviewThinkingLevel(level) {
		issues = append(issues, ValidationIssue{
			Field:      "review.thinking.level",
			Value:      level,
			Message:    "review.thinking.level は空、または low/medium/high/xhigh のいずれかを指定してください",
			Suggestion: "",
			Severity:   ValidationSeverityError,
			CanAutoFix: false,
		})
	}
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
