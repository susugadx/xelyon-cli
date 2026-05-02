package config

import (
	"fmt"
	"strings"
)

var validBashSafetyLevels = []string{"strict", "moderate", "permissive"}

func validateBashSafetyLevelIssues(cfg *Config) []ValidationIssue {
	level := cfg.Bash.SafetyLevel
	if level == "" || contains(validBashSafetyLevels, level) {
		return nil
	}
	return []ValidationIssue{{
		Field:      "bash.safety_level",
		Value:      level,
		Message:    fmt.Sprintf("無効な安全性レベルです (有効: %s)", strings.Join(validBashSafetyLevels, ", ")),
		Suggestion: "moderate",
		Severity:   ValidationSeverityWarning,
		CanAutoFix: true,
		FixedValue: "moderate",
	}}
}
