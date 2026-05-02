package config

import (
	"fmt"
	"strconv"
)

func validateAgentInstructionIssues(cfg *Config) []ValidationIssue {
	var issues []ValidationIssue
	if cfg == nil {
		return issues
	}

	if !isValidAgentInstructionProjectMode(cfg.AgentInstructions.Project.Mode) {
		issues = append(issues, ValidationIssue{
			Field:      "agent_instructions.project.mode",
			Value:      cfg.AgentInstructions.Project.Mode,
			Message:    fmt.Sprintf("無効な値です (有効: %s / %s / %s)", AgentInstructionProjectModeOff, AgentInstructionProjectModeFallback, AgentInstructionProjectModeAlways),
			Suggestion: AgentInstructionProjectModeFallback,
			Severity:   ValidationSeverityWarning,
			CanAutoFix: true,
			FixedValue: AgentInstructionProjectModeFallback,
		})
	}

	if issue, ok := validatePositiveIntIssue("agent_instructions.max_file_bytes", cfg.AgentInstructions.MaxFileBytes, DefaultConfig().AgentInstructions.MaxFileBytes); ok {
		issues = append(issues, issue)
	}
	if issue, ok := validatePositiveIntIssue("agent_instructions.max_total_bytes", cfg.AgentInstructions.MaxTotalBytes, DefaultConfig().AgentInstructions.MaxTotalBytes); ok {
		issues = append(issues, issue)
	}

	return issues
}

func validatePositiveIntIssue(field string, value int, defaultVal int) (ValidationIssue, bool) {
	if value > 0 {
		return ValidationIssue{}, false
	}
	return ValidationIssue{
		Field:      field,
		Value:      strconv.Itoa(value),
		Message:    "0 より大きい値を指定してください",
		Suggestion: fmt.Sprintf("%d", defaultVal),
		Severity:   ValidationSeverityWarning,
		CanAutoFix: true,
		FixedValue: defaultVal,
	}, true
}
