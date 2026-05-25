package config

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

type geminiFunctionCallingAutoFix struct {
	provider      string
	target        string
	overrideModel string
	replacement   string
}

const (
	geminiFunctionCallingFixGlobalDefaultModel   = "global_default_model"
	geminiFunctionCallingFixDefaultModel         = "default_model"
	geminiFunctionCallingFixCatalogModel         = "catalog_model"
	geminiFunctionCallingFixOverrideCatalogModel = "model_override_catalog_model"
)

func validateGeminiFunctionCallingIssues(cfg *Config) []ValidationIssue {
	var issues []ValidationIssue
	if globalIssue := validateGeminiGlobalDefaultModelIssue(cfg); globalIssue != nil {
		issues = append(issues, *globalIssue)
	}

	source, providerKey, ok := cfg.explicitProviderModelSelection("gemini")
	if !ok {
		return issues
	}
	pm, ok := explicitSelectedProviderModelConfig(source, providerKey)
	if !ok {
		return issues
	}

	defaultModel := strings.TrimSpace(pm.DefaultModel)
	catalogModel := strings.TrimSpace(pm.CatalogModel)
	if defaultModel != "" {
		policy := llmcatalog.NewGeminiFunctionCallingPolicy(defaultModel, catalogModel)
		if geminiFunctionCallingUnsupported(policy.RequestSupport()) {
			issues = append(issues, geminiFunctionCallingAutoFixIssue(
				fmt.Sprintf("provider_models.%s.default_model", providerKey),
				defaultModel,
				policy.RequestSupport(),
				geminiFunctionCallingAutoFix{
					provider: providerKey,
					target:   geminiFunctionCallingFixDefaultModel,
				},
			))
		}
	}
	if catalogModel != "" {
		policy := llmcatalog.NewGeminiFunctionCallingPolicy(defaultModel, catalogModel)
		if geminiFunctionCallingUnsupported(policy.CatalogSupport()) {
			issues = append(issues, geminiFunctionCallingAutoFixIssue(
				fmt.Sprintf("provider_models.%s.catalog_model", providerKey),
				catalogModel,
				policy.CatalogSupport(),
				geminiFunctionCallingAutoFix{
					provider: providerKey,
					target:   geminiFunctionCallingFixCatalogModel,
				},
			))
		}
	}

	for overrideModel, override := range pm.ModelOverrides {
		catalogModel := strings.TrimSpace(override.CatalogModel)
		policy := llmcatalog.NewGeminiFunctionCallingPolicy(overrideModel, catalogModel)
		if geminiFunctionCallingUnsupported(policy.RequestSupport()) {
			issues = append(issues, geminiFunctionCallingManualIssue(
				fmt.Sprintf("provider_models.%s.model_overrides.%s", providerKey, overrideModel),
				overrideModel,
				policy.RequestSupport(),
			))
			continue
		}
		if catalogModel == "" {
			continue
		}
		if geminiFunctionCallingUnsupported(policy.CatalogSupport()) {
			field := fmt.Sprintf("provider_models.%s.model_overrides.%s.catalog_model", providerKey, overrideModel)
			issues = append(issues, geminiFunctionCallingAutoFixIssue(field, catalogModel, policy.CatalogSupport(), geminiFunctionCallingAutoFix{
				provider:      providerKey,
				target:        geminiFunctionCallingFixOverrideCatalogModel,
				overrideModel: overrideModel,
			}))
		}
	}
	return issues
}

func validateGeminiGlobalDefaultModelIssue(cfg *Config) *ValidationIssue {
	if cfg == nil || !SameProviderRuntimeIdentity(cfg.DefaultProvider, "gemini") {
		return nil
	}
	defaultModel := strings.TrimSpace(cfg.DefaultModel)
	if defaultModel == "" || !cfg.configuredDefaultModelAppliesToProvider("gemini", defaultModel) {
		return nil
	}
	if pm, ok := cfg.rawExplicitProviderModelConfig("gemini"); ok && strings.TrimSpace(pm.DefaultModel) != "" {
		return nil
	}

	policy := llmcatalog.NewGeminiFunctionCallingPolicy(defaultModel, "")
	if !geminiFunctionCallingUnsupported(policy.RequestSupport()) {
		return nil
	}
	issue := geminiFunctionCallingAutoFixIssue("default_model", defaultModel, policy.RequestSupport(), geminiFunctionCallingAutoFix{
		target: geminiFunctionCallingFixGlobalDefaultModel,
	})
	return &issue
}

func geminiFunctionCallingUnsupported(support llmcatalog.ModelCapabilitySupport) bool {
	return support.Known && !support.Supported
}

func geminiFunctionCallingAutoFixIssue(field string, value string, support llmcatalog.ModelCapabilitySupport, fix geminiFunctionCallingAutoFix) ValidationIssue {
	replacement := geminiFunctionCallingReplacement(support)
	fix.replacement = replacement
	return ValidationIssue{
		Field:      field,
		Value:      value,
		Message:    "Gemini model は XELYON の native function calling runtime に対応している必要があります",
		Suggestion: fmt.Sprintf("%s を %q に変更してください", field, replacement),
		Severity:   ValidationSeverityError,
		CanAutoFix: true,
		FixedValue: fix,
	}
}

func geminiFunctionCallingManualIssue(field string, value string, support llmcatalog.ModelCapabilitySupport) ValidationIssue {
	replacement := geminiFunctionCallingReplacement(support)
	return ValidationIssue{
		Field:      field,
		Value:      value,
		Message:    "Gemini model は XELYON の native function calling runtime に対応している必要があります",
		Suggestion: fmt.Sprintf("%s は map key のため自動修正できません。%q に変更するか、この override を削除してください", field, replacement),
		Severity:   ValidationSeverityError,
		CanAutoFix: false,
	}
}

func geminiFunctionCallingReplacement(support llmcatalog.ModelCapabilitySupport) string {
	if replacement := strings.TrimSpace(support.Replacement); replacement != "" {
		return replacement
	}
	return "gemini-3.5-flash"
}

func applyGeminiFunctionCallingAutoFix(cfg *Config, value any) bool {
	fix, ok := value.(geminiFunctionCallingAutoFix)
	if !ok || cfg == nil || fix.replacement == "" {
		return false
	}
	if !fix.validTarget() {
		return false
	}
	if fix.target == geminiFunctionCallingFixGlobalDefaultModel {
		cfg.DefaultModel = fix.replacement
		return true
	}

	return cfg.PatchProviderModelConfig(fix.provider, func(pm *ProviderModelConfig) {
		switch fix.target {
		case geminiFunctionCallingFixDefaultModel:
			pm.DefaultModel = fix.replacement
		case geminiFunctionCallingFixCatalogModel:
			pm.CatalogModel = fix.replacement
		case geminiFunctionCallingFixOverrideCatalogModel:
			if pm.ModelOverrides == nil {
				return
			}
			override := pm.ModelOverrides[fix.overrideModel]
			override.CatalogModel = fix.replacement
			pm.ModelOverrides[fix.overrideModel] = override
		}
	})
}

func (fix geminiFunctionCallingAutoFix) validTarget() bool {
	switch fix.target {
	case geminiFunctionCallingFixGlobalDefaultModel:
		return true
	case geminiFunctionCallingFixDefaultModel, geminiFunctionCallingFixCatalogModel:
		return fix.provider != ""
	case geminiFunctionCallingFixOverrideCatalogModel:
		return fix.provider != "" && fix.overrideModel != ""
	default:
		return false
	}
}
