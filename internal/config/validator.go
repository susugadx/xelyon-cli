package config

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// ValidProviders は有効なプロバイダー名の一覧
// internal/api/provider.go の NewProvider() と同期させること
var ValidProviders = []string{
	"deepseek",
	"openai",
	"gemini",
	"claude",
	"anthropic", // claudeのエイリアス
	"ollama",
	"groq",
	"openrouter",
	"bedrock",
}

// GetDisplayProviders は表示用のプロバイダーリスト（エイリアスを除く）を返す
func GetDisplayProviders() []string {
	var display []string
	for _, p := range ValidProviders {
		if p == "anthropic" {
			continue
		}
		display = append(display, p)
	}
	return display
}

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

	// 1. 必須項目チェック
	if cfg.DefaultProvider == "" {
		result.Issues = append(result.Issues, ValidationIssue{
			Field:      "default_provider",
			Value:      "",
			Message:    "必須項目が未設定です",
			Suggestion: "deepseek",
			Severity:   "error",
			CanAutoFix: true,
			FixedValue: "deepseek",
		})
		result.Valid = false
	}

	// 2. プロバイダー名検証
	if cfg.DefaultProvider != "" && !isValidProvider(cfg.DefaultProvider) {
		result.Issues = append(result.Issues, ValidationIssue{
			Field:      "default_provider",
			Value:      cfg.DefaultProvider,
			Message:    fmt.Sprintf("無効なプロバイダー名です (有効: %s)", strings.Join(ValidProviders, ", ")),
			Suggestion: suggestProvider(cfg.DefaultProvider),
			Severity:   "error",
			CanAutoFix: true,
			FixedValue: suggestProvider(cfg.DefaultProvider),
		})
		result.Valid = false
	}

	// 3. ProviderModels のプロバイダー名検証
	for providerName := range cfg.ProviderModels {
		if !isValidProvider(providerName) {
			result.Issues = append(result.Issues, ValidationIssue{
				Field:      fmt.Sprintf("provider_models.%s", providerName),
				Value:      providerName,
				Message:    "無効なプロバイダー名です",
				Suggestion: fmt.Sprintf("削除するか、有効なプロバイダー名に変更してください (%s)", strings.Join(ValidProviders, ", ")),
				Severity:   "warning",
				CanAutoFix: false,
			})
		}
	}

	// 4. 数値範囲チェック
	validateNumericRange(&result, "loop_detection.threshold", cfg.LoopDetection.Threshold, 1, 10, 3)
	validateNumericRange(&result, "api_retry.count", cfg.APIRetry.Count, 1, 10, 3)
	validateNumericRange(&result, "api_retry.initial_delay", cfg.APIRetry.InitialDelay, 1, 60, 1)
	validateNumericRange(&result, "api_retry.max_delay", cfg.APIRetry.MaxDelay, 1, 300, 30)
	validateNumericRange(&result, "api_retry.timeout", cfg.APIRetry.Timeout, 30, 7200, 3600)
	validateNumericRange(&result, "backup.max_generations", cfg.Backup.MaxGenerations, 1, 20, 5)
	validateNumericRange(&result, "compression.keep_recent", cfg.Compression.KeepRecent, 1, 100, 10)
	validateNumericRange(&result, "diff.context_lines", cfg.Diff.ContextLines, 0, 100, 10)
	validateNumericRange(&result, "code_health.max_file_lines", cfg.CodeHealth.MaxFileLines, 100, 2000, 300)
	validateNumericRange(&result, "code_health.max_function_lines", cfg.CodeHealth.MaxFunctionLines, 10, 500, 50)
	validateNumericRange(&result, "paste.max_lines", cfg.Paste.MaxLines, 100, 100000, 10000)
	validateNumericRange(&result, "paste.timeout_seconds", cfg.Paste.TimeoutSeconds, 10, 600, 60)
	validateNumericRange(&result, "streaming.idle_timeout_seconds", cfg.Streaming.IdleTimeoutSeconds, 10, 7200, 3600)

	// 5. Bash安全性レベル検証
	if cfg.Bash.SafetyLevel != "" {
		validLevels := []string{"strict", "moderate", "permissive"}
		if !contains(validLevels, cfg.Bash.SafetyLevel) {
			result.Issues = append(result.Issues, ValidationIssue{
				Field:      "bash.safety_level",
				Value:      cfg.Bash.SafetyLevel,
				Message:    fmt.Sprintf("無効な安全性レベルです (有効: %s)", strings.Join(validLevels, ", ")),
				Suggestion: "moderate",
				Severity:   "warning",
				CanAutoFix: true,
				FixedValue: "moderate",
			})
		}
	}

	return result
}

// validateNumericRange は数値の範囲をチェックし、問題があればIssueを追加
func validateNumericRange(result *ValidationResult, field string, value, min, max, defaultVal int) {
	if value == 0 {
		return // 未設定はデフォルト適用されるためスキップ
	}
	if value < min || value > max {
		result.Issues = append(result.Issues, ValidationIssue{
			Field:      field,
			Value:      fmt.Sprintf("%d", value),
			Message:    fmt.Sprintf("推奨範囲外です (推奨: %d-%d)", min, max),
			Suggestion: fmt.Sprintf("%d", defaultVal),
			Severity:   "warning",
			CanAutoFix: true,
			FixedValue: defaultVal,
		})
	}
}

// isValidProvider はプロバイダー名が有効かチェック
func isValidProvider(name string) bool {
	lower := strings.ToLower(name)
	for _, valid := range ValidProviders {
		if lower == valid {
			return true
		}
	}
	return false
}

// suggestProvider は類似のプロバイダー名を提案
func suggestProvider(invalid string) string {
	lower := strings.ToLower(invalid)

	// 完全一致チェック（大文字小文字違いの場合）
	for _, valid := range ValidProviders {
		if lower == valid {
			return valid
		}
	}

	// 部分一致で提案
	for _, valid := range ValidProviders {
		if strings.Contains(lower, valid) || strings.Contains(valid, lower) {
			return valid
		}
	}

	// よくあるタイプミス
	typoMap := map[string]string{
		"deepseak":  "deepseek",
		"deepseek2": "deepseek",
		"gpt":       "openai",
		"chatgpt":   "openai",
		"gpt4":      "openai",
		"google":    "gemini",
		"palm":      "gemini",
		"bard":      "gemini",
		"sonnet":    "claude",
		"opus":      "claude",
		"haiku":     "claude",
		"llama":     "ollama",
		"mistral":   "ollama",
		"qwen":      "ollama",
		"aws":       "bedrock",
		"amazon":    "bedrock",
	}
	if suggestion, ok := typoMap[lower]; ok {
		return suggestion
	}

	return "deepseek" // デフォルト提案
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

// PrintValidationWarnings はバリデーション警告を表示
func PrintValidationWarnings(result ValidationResult) {
	if len(result.Issues) == 0 {
		return
	}

	yellow.Println("\n⚠️  設定ファイルに問題があります: ~/.xelyon/config.yaml")
	fmt.Println()

	for i, issue := range result.Issues {
		icon := "⚠️"
		if issue.Severity == "error" {
			icon = "❌"
		}

		fmt.Printf("%d. %s %s: %s\n", i+1, icon, issue.Field, issue.Message)
		if issue.Value != "" {
			fmt.Printf("   現在の値: %s\n", issue.Value)
		}
		if issue.Suggestion != "" {
			cyan.Printf("   提案: %s\n", issue.Suggestion)
		}
		fmt.Println()
	}
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
		case "loop_detection.threshold":
			if v, ok := issue.FixedValue.(int); ok {
				cfg.LoopDetection.Threshold = v
				fixCount++
			}
		case "api_retry.count":
			if v, ok := issue.FixedValue.(int); ok {
				cfg.APIRetry.Count = v
				fixCount++
			}
		case "api_retry.initial_delay":
			if v, ok := issue.FixedValue.(int); ok {
				cfg.APIRetry.InitialDelay = v
				fixCount++
			}
		case "api_retry.max_delay":
			if v, ok := issue.FixedValue.(int); ok {
				cfg.APIRetry.MaxDelay = v
				fixCount++
			}
		case "api_retry.timeout":
			if v, ok := issue.FixedValue.(int); ok {
				cfg.APIRetry.Timeout = v
				fixCount++
			}
		case "backup.max_generations":
			if v, ok := issue.FixedValue.(int); ok {
				cfg.Backup.MaxGenerations = v
				fixCount++
			}
		case "compression.keep_recent":
			if v, ok := issue.FixedValue.(int); ok {
				cfg.Compression.KeepRecent = v
				fixCount++
			}
		case "diff.context_lines":
			if v, ok := issue.FixedValue.(int); ok {
				cfg.Diff.ContextLines = v
				fixCount++
			}
		case "code_health.max_file_lines":
			if v, ok := issue.FixedValue.(int); ok {
				cfg.CodeHealth.MaxFileLines = v
				fixCount++
			}
		case "code_health.max_function_lines":
			if v, ok := issue.FixedValue.(int); ok {
				cfg.CodeHealth.MaxFunctionLines = v
				fixCount++
			}
		case "paste.max_lines":
			if v, ok := issue.FixedValue.(int); ok {
				cfg.Paste.MaxLines = v
				fixCount++
			}
		case "paste.timeout_seconds":
			if v, ok := issue.FixedValue.(int); ok {
				cfg.Paste.TimeoutSeconds = v
				fixCount++
			}
		case "streaming.idle_timeout_seconds":
			if v, ok := issue.FixedValue.(int); ok {
				cfg.Streaming.IdleTimeoutSeconds = v
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
