package config

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

// ValidProviders は有効なプロバイダー名の一覧
var ValidProviders = llmcatalog.ProviderKeys(true)

var providerTypos = map[string]string{
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

// GetDisplayProviders は表示用のプロバイダーリスト（エイリアスを除く）を返す
func GetDisplayProviders() []string {
	return llmcatalog.DisplayProviderKeys()
}

func validateProviderIssues(cfg *Config) []ValidationIssue {
	var issues []ValidationIssue
	if issue, ok := validateRequiredProviderIssue(cfg); ok {
		issues = append(issues, issue)
	}
	if issue, ok := validateDefaultProviderIssue(cfg); ok {
		issues = append(issues, issue)
	}
	issues = append(issues, validateProviderModelsIssues(cfg)...)
	return issues
}

func validateRequiredProviderIssue(cfg *Config) (ValidationIssue, bool) {
	if cfg.DefaultProvider != "" {
		return ValidationIssue{}, false
	}
	return ValidationIssue{
		Field:      "default_provider",
		Value:      "",
		Message:    "必須項目が未設定です",
		Suggestion: "deepseek",
		Severity:   "error",
		CanAutoFix: true,
		FixedValue: "deepseek",
	}, true
}

func validateDefaultProviderIssue(cfg *Config) (ValidationIssue, bool) {
	if cfg.DefaultProvider == "" || isValidProvider(cfg.DefaultProvider) {
		return ValidationIssue{}, false
	}
	suggested := suggestProvider(cfg.DefaultProvider)
	return ValidationIssue{
		Field:      "default_provider",
		Value:      cfg.DefaultProvider,
		Message:    fmt.Sprintf("無効なプロバイダー名です (有効: %s)", strings.Join(ValidProviders, ", ")),
		Suggestion: suggested,
		Severity:   "error",
		CanAutoFix: true,
		FixedValue: suggested,
	}, true
}

func validateProviderModelsIssues(cfg *Config) []ValidationIssue {
	var issues []ValidationIssue
	validProviders := strings.Join(ValidProviders, ", ")
	for providerName := range cfg.ProviderModelsForEdit() {
		if isValidProvider(providerName) {
			continue
		}
		issues = append(issues, ValidationIssue{
			Field:      fmt.Sprintf("provider_models.%s", providerName),
			Value:      providerName,
			Message:    "無効なプロバイダー名です",
			Suggestion: fmt.Sprintf("削除するか、有効なプロバイダー名に変更してください (%s)", validProviders),
			Severity:   "warning",
			CanAutoFix: false,
		})
	}
	return issues
}

// isValidProvider はプロバイダー名が有効かチェック
func isValidProvider(name string) bool {
	return llmcatalog.IsKnownProvider(name)
}

// suggestProvider は類似のプロバイダー名を提案
func suggestProvider(invalid string) string {
	lower := NormalizeProviderName(invalid)

	for _, valid := range ValidProviders {
		if lower == valid {
			return valid
		}
	}
	for _, valid := range ValidProviders {
		if strings.Contains(lower, valid) || strings.Contains(valid, lower) {
			return valid
		}
	}
	if suggestion, ok := providerTypos[lower]; ok {
		return suggestion
	}

	return "deepseek"
}
