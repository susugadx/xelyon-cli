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
	Severity   string // "error" or "warning"
	CanAutoFix bool   // 自動修正可能か
	FixedValue any    // 自動修正後の値
}

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

	return result
}

func appendValidationIssues(result *ValidationResult, issues []ValidationIssue) {
	if len(issues) == 0 {
		return
	}
	result.Issues = append(result.Issues, issues...)
	for _, issue := range issues {
		if issue.Severity == "error" {
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
		if issue.Severity == "error" {
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

	for _, issue := range result.Issues {
		if !issue.CanAutoFix || issue.FixedValue == nil {
			continue
		}

		switch issue.Field {
		case "default_provider":
			if v, ok := issue.FixedValue.(string); ok {
				cfg.DefaultProvider = v
				fixCount++
			}
		case "compression.trigger_percent":
			if v, ok := issue.FixedValue.(int); ok {
				cfg.Compression.TriggerPercent = v
				fixCount++
			}
		case "compression.keep_recent":
			if v, ok := issue.FixedValue.(int); ok {
				cfg.Compression.KeepRecent = v
				fixCount++
			}
		case "project_map.context_ratio":
			if v, ok := issue.FixedValue.(float64); ok {
				cfg.ProjectMap.ContextRatio = v
				fixCount++
			}
		case "bash.safety_level":
			if v, ok := issue.FixedValue.(string); ok {
				cfg.Bash.SafetyLevel = v
				fixCount++
			}
		}
	}

	return fixCount
}

// HasErrors は致命的なエラーがあるかチェック
func (r ValidationResult) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Severity == "error" {
			return true
		}
	}
	return false
}

// HasWarnings は警告があるかチェック
func (r ValidationResult) HasWarnings() bool {
	for _, issue := range r.Issues {
		if issue.Severity == "warning" {
			return true
		}
	}
	return false
}
