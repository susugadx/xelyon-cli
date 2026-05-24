package config

import (
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/stdio"
)

// ValidationIssue はバリデーション問題を表す
type ValidationIssue struct {
	Field      string // 問題のあるフィールド名
	Value      string // 現在の値
	Message    string // 問題の説明
	Suggestion string // 修正提案
	Severity   ValidationSeverity
	CanAutoFix bool // 自動修正可能か
	FixedValue any  // 自動修正後の値
}

// ValidationSeverity は validation issue の深刻度。
type ValidationSeverity string

const (
	ValidationSeverityError   ValidationSeverity = "error"
	ValidationSeverityWarning ValidationSeverity = "warning"
)

// ValidationResult はバリデーション結果を表す
type ValidationResult struct {
	Valid  bool
	Issues []ValidationIssue
}

var (
	yellow = color.New(color.FgYellow)
	cyan   = color.New(color.FgCyan)
)

// ValidateConfig は設定ファイルのバリデーションを行う
func ValidateConfig(cfg *Config) ValidationResult {
	result := ValidationResult{Valid: true}

	appendValidationIssues(&result, validateProviderIssues(cfg))
	appendValidationIssues(&result, validateNumericRangeIssues(cfg))
	appendValidationIssues(&result, validateBashSafetyLevelIssues(cfg))
	appendValidationIssues(&result, validateAgentInstructionIssues(cfg))
	appendValidationIssues(&result, validateGeminiIssues(cfg))

	return result
}

func appendValidationIssues(result *ValidationResult, issues []ValidationIssue) {
	if len(issues) == 0 {
		return
	}
	result.Issues = append(result.Issues, issues...)
	for _, issue := range issues {
		if issue.Severity == ValidationSeverityError {
			result.Valid = false
		}
	}
}

// contains は文字列スライスに指定文字列が含まれるかチェック
func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// PrintValidationWarningsToWriter は指定 writer にバリデーション警告を表示する。
func PrintValidationWarningsToWriter(w io.Writer, result ValidationResult) {
	if len(result.Issues) == 0 {
		return
	}
	if w == nil {
		w = stdio.Output()
	}

	yellow.Fprintln(w, "\n⚠️  設定ファイルに問題があります: ~/.xelyon/config.yaml")
	fmt.Fprintln(w)

	for i, issue := range result.Issues {
		icon := "⚠️"
		if issue.Severity == ValidationSeverityError {
			icon = "❌"
		}

		fmt.Fprintf(w, "%d. %s %s: %s\n", i+1, icon, issue.Field, issue.Message)
		if issue.Value != "" {
			fmt.Fprintf(w, "   現在の値: %s\n", issue.Value)
		}
		if issue.Suggestion != "" {
			cyan.Fprintf(w, "   提案: %s\n", issue.Suggestion)
		}
		fmt.Fprintln(w)
	}
}

// PrintValidationWarnings はバリデーション警告を表示
func PrintValidationWarnings(result ValidationResult) {
	PrintValidationWarningsToWriter(nil, result)
}

// ApplyAutoFixes は自動修正可能な問題を修正
func ApplyAutoFixes(cfg *Config, result ValidationResult) int {
	fixCount := 0
	if cfg == nil {
		return fixCount
	}

	for _, issue := range result.Issues {
		if !issue.CanAutoFix || issue.FixedValue == nil {
			continue
		}
		fixer, ok := configAutoFixers[issue.Field]
		if !ok {
			continue
		}
		if fixer(cfg, issue.FixedValue) {
			fixCount++
		}
	}

	return fixCount
}

type configAutoFixer func(*Config, any) bool

var configAutoFixers = map[string]configAutoFixer{
	"default_provider": stringAutoFixer(func(cfg *Config, v string) { cfg.DefaultProvider = v }),
	"compression.trigger_percent": intAutoFixer(func(cfg *Config, v int) {
		cfg.Compression.TriggerPercent = v
	}),
	"compression.keep_recent": intAutoFixer(func(cfg *Config, v int) {
		cfg.Compression.KeepRecent = v
	}),
	"project_map.context_ratio": floatAutoFixer(func(cfg *Config, v float64) {
		cfg.ProjectMap.ContextRatio = v
	}),
	"bash.safety_level": stringAutoFixer(func(cfg *Config, v string) {
		cfg.Bash.SafetyLevel = v
	}),
	"gemini.service_tier": stringAutoFixer(func(cfg *Config, v string) {
		cfg.Gemini.ServiceTier = v
	}),
	"agent_instructions.project.mode": stringAutoFixer(func(cfg *Config, v string) {
		cfg.AgentInstructions.Project.Mode = v
	}),
	"agent_instructions.max_file_bytes": intAutoFixer(func(cfg *Config, v int) {
		cfg.AgentInstructions.MaxFileBytes = v
	}),
	"agent_instructions.max_total_bytes": intAutoFixer(func(cfg *Config, v int) {
		cfg.AgentInstructions.MaxTotalBytes = v
	}),
}

func stringAutoFixer(setter func(*Config, string)) configAutoFixer {
	return func(cfg *Config, value any) bool {
		typed, ok := value.(string)
		if !ok {
			return false
		}
		setter(cfg, typed)
		return true
	}
}

func intAutoFixer(setter func(*Config, int)) configAutoFixer {
	return func(cfg *Config, value any) bool {
		typed, ok := value.(int)
		if !ok {
			return false
		}
		setter(cfg, typed)
		return true
	}
}

func floatAutoFixer(setter func(*Config, float64)) configAutoFixer {
	return func(cfg *Config, value any) bool {
		typed, ok := value.(float64)
		if !ok {
			return false
		}
		setter(cfg, typed)
		return true
	}
}

// HasErrors は致命的なエラーがあるかチェック
func (r ValidationResult) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Severity == ValidationSeverityError {
			return true
		}
	}
	return false
}

// HasWarnings は警告があるかチェック
func (r ValidationResult) HasWarnings() bool {
	for _, issue := range r.Issues {
		if issue.Severity == ValidationSeverityWarning {
			return true
		}
	}
	return false
}
