package config

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

// GeminiFunctionCallingPolicyForModel は config の catalog_model 解決を反映した
// Gemini function calling policy を返す。
func GeminiFunctionCallingPolicyForModel(cfg *Config, provider, model string) llmcatalog.GeminiFunctionCallingPolicy {
	model = strings.TrimSpace(model)
	if cfg == nil {
		cfg = DefaultConfig()
	}
	catalogModel := cfg.ModelCatalogName(provider, model)
	return llmcatalog.NewGeminiFunctionCallingPolicy(model, catalogModel)
}

// ValidateGeminiFunctionCallingConfig は config 内の Gemini model 設定が
// native function calling runtime で利用可能か検証する。
func ValidateGeminiFunctionCallingConfig(cfg *Config) ValidationResult {
	result := ValidationResult{Valid: true}
	appendValidationIssues(&result, validateGeminiFunctionCallingIssues(cfg))
	return result
}

// ValidateGeminiFunctionCallingSelection は Gemini の選択 model が
// native function calling runtime で利用可能か検証する。
func ValidateGeminiFunctionCallingSelection(cfg *Config, provider, model string) error {
	if CanonicalProviderName(provider) != "gemini" {
		return nil
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return nil
	}
	return GeminiFunctionCallingPolicyForModel(cfg, provider, model).UnsupportedError()
}
