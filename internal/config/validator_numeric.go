package config

import (
	"fmt"
	"math"
	"strconv"
)

func validateNumericRangeIssues(cfg *Config) []ValidationIssue {
	var issues []ValidationIssue
	if issue, ok := validateNumericRangeIssue("compression.trigger_percent", cfg.Compression.TriggerPercent, 1, 100, 80); ok {
		issues = append(issues, issue)
	}
	if issue, ok := validateNumericRangeIssue("compression.keep_recent", cfg.Compression.KeepRecent, 1, 100, 10); ok {
		issues = append(issues, issue)
	}
	if issue, ok := validateProjectMapContextRatioIssue(cfg.ProjectMap.ContextRatio); ok {
		issues = append(issues, issue)
	}
	if issue, ok := validateResponsesServerCompactionCompactThresholdIssue(cfg.Responses.ServerCompaction.CompactThreshold); ok {
		issues = append(issues, issue)
	}
	return issues
}

func validateNumericRangeIssue(field string, value, min, max, defaultVal int) (ValidationIssue, bool) {
	if value == 0 {
		return ValidationIssue{}, false
	}
	if value >= min && value <= max {
		return ValidationIssue{}, false
	}
	return ValidationIssue{
		Field:      field,
		Value:      fmt.Sprintf("%d", value),
		Message:    fmt.Sprintf("推奨範囲外です (推奨: %d-%d)", min, max),
		Suggestion: fmt.Sprintf("%d", defaultVal),
		Severity:   ValidationSeverityWarning,
		CanAutoFix: true,
		FixedValue: defaultVal,
	}, true
}

func validateProjectMapContextRatioIssue(value float64) (ValidationIssue, bool) {
	if !math.IsNaN(value) && !math.IsInf(value, 0) && value >= ProjectMapContextRatioMin && value <= ProjectMapContextRatioMax {
		return ValidationIssue{}, false
	}
	return ValidationIssue{
		Field:      "project_map.context_ratio",
		Value:      formatFloatValidationValue(value),
		Message:    fmt.Sprintf("有効範囲外です (有効: %.2f-%.2f)", ProjectMapContextRatioMin, ProjectMapContextRatioMax),
		Suggestion: fmt.Sprintf("%.2f", ProjectMapContextRatioDefault),
		Severity:   ValidationSeverityWarning,
		CanAutoFix: true,
		FixedValue: ProjectMapContextRatioDefault,
	}, true
}

func validateResponsesServerCompactionCompactThresholdIssue(value int) (ValidationIssue, bool) {
	if value <= 0 || value >= 1000 {
		return ValidationIssue{}, false
	}
	return ValidationIssue{
		Field:      "responses.server_compaction.compact_threshold",
		Value:      fmt.Sprintf("%d", value),
		Message:    "compact_threshold は 0（auto）または 1000 以上を指定してください",
		Suggestion: "0",
		Severity:   ValidationSeverityError,
		CanAutoFix: false,
	}, true
}

func formatFloatValidationValue(value float64) string {
	switch {
	case math.IsNaN(value):
		return "NaN"
	case math.IsInf(value, 1):
		return "+Inf"
	case math.IsInf(value, -1):
		return "-Inf"
	default:
		return strconv.FormatFloat(value, 'g', -1, 64)
	}
}
