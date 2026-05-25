package config

import (
	"strings"
	"testing"
)

func TestValidateGeminiFunctionCallingSelection(t *testing.T) {
	cfg := DefaultConfig()
	if err := ValidateGeminiFunctionCallingSelection(cfg, "gemini", "gemini-3.5-flash"); err != nil {
		t.Fatalf("supported Gemini model error = %v, want nil", err)
	}
	if err := ValidateGeminiFunctionCallingSelection(cfg, "gemini", "corp-gemini"); err != nil {
		t.Fatalf("unknown Gemini alias error = %v, want nil", err)
	}

	err := ValidateGeminiFunctionCallingSelection(cfg, "gemini", "gemini-2.0-flash-lite")
	if err == nil || !strings.Contains(err.Error(), "gemini-3.1-flash-lite") {
		t.Fatalf("unsupported Gemini model error = %v, want replacement guidance", err)
	}

	cfg.SetProviderModelConfig("gemini", ProviderModelConfig{
		ModelOverrides: map[string]ModelOverride{
			"corp-lite": {CatalogModel: "gemini-2.0-flash-lite"},
		},
	})
	err = ValidateGeminiFunctionCallingSelection(cfg, "gemini", "corp-lite")
	if err == nil || !strings.Contains(err.Error(), "corp-lite (catalog_model=gemini-2.0-flash-lite)") {
		t.Fatalf("unsupported Gemini catalog_model error = %v, want alias and catalog_model detail", err)
	}
}

func TestValidateConfig_GeminiFunctionCallingProviderModels(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"gemini": {
			DefaultModel: "gemini-2.0-flash-lite",
			ModelOverrides: map[string]ModelOverride{
				"corp-lite": {CatalogModel: "models/gemini-2.0-flash-lite"},
			},
		},
	})

	result := ValidateConfig(cfg)
	defaultIssue := findValidationIssue(result, "provider_models.gemini.default_model")
	if defaultIssue == nil || defaultIssue.Severity != ValidationSeverityError || !defaultIssue.CanAutoFix {
		t.Fatalf("default_model issue = %#v, want autofixable error", defaultIssue)
	}
	overrideIssue := findValidationIssue(result, "provider_models.gemini.model_overrides.corp-lite.catalog_model")
	if overrideIssue == nil || overrideIssue.Severity != ValidationSeverityError || !overrideIssue.CanAutoFix {
		t.Fatalf("override catalog_model issue = %#v, want autofixable error", overrideIssue)
	}
	if result.Valid {
		t.Fatal("ValidateConfig() Valid = true, want false for unsupported Gemini function calling model")
	}

	if fixed := ApplyAutoFixes(cfg, result); fixed != 2 {
		t.Fatalf("ApplyAutoFixes() = %d, want 2", fixed)
	}
	pm, ok := cfg.rawExplicitProviderModelConfig("gemini")
	if !ok {
		t.Fatal("provider_models.gemini should remain configured")
	}
	if pm.DefaultModel != "gemini-3.1-flash-lite" {
		t.Fatalf("DefaultModel = %q, want gemini-3.1-flash-lite", pm.DefaultModel)
	}
	if got := pm.ModelOverrides["corp-lite"].CatalogModel; got != "gemini-3.1-flash-lite" {
		t.Fatalf("override CatalogModel = %q, want gemini-3.1-flash-lite", got)
	}
}

func TestValidateConfig_GeminiCatalogModelAutoFix(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"gemini": {
			DefaultModel: "corp-lite",
			CatalogModel: "models/gemini-2.0-flash-lite",
		},
	})

	result := ValidateConfig(cfg)
	issue := findValidationIssue(result, "provider_models.gemini.catalog_model")
	if issue == nil || issue.Severity != ValidationSeverityError || !issue.CanAutoFix {
		t.Fatalf("catalog_model issue = %#v, want autofixable error", issue)
	}
	if fixed := ApplyAutoFixes(cfg, result); fixed != 1 {
		t.Fatalf("ApplyAutoFixes() = %d, want 1", fixed)
	}
	pm, _ := cfg.rawExplicitProviderModelConfig("gemini")
	if pm.CatalogModel != "gemini-3.1-flash-lite" {
		t.Fatalf("CatalogModel = %q, want gemini-3.1-flash-lite", pm.CatalogModel)
	}
}

func TestValidateConfig_GeminiCatalogModelOnlyAutoFix(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"gemini": {
			CatalogModel: "models/gemini-2.0-flash-lite",
		},
	})

	result := ValidateConfig(cfg)
	issue := findValidationIssue(result, "provider_models.gemini.catalog_model")
	if issue == nil || issue.Severity != ValidationSeverityError || !issue.CanAutoFix {
		t.Fatalf("catalog_model issue = %#v, want autofixable error", issue)
	}
	if result.Valid {
		t.Fatal("ValidateConfig() Valid = true, want false for unsupported Gemini catalog_model")
	}
	if fixed := ApplyAutoFixes(cfg, result); fixed != 1 {
		t.Fatalf("ApplyAutoFixes() = %d, want 1", fixed)
	}
	pm, _ := cfg.rawExplicitProviderModelConfig("gemini")
	if pm.CatalogModel != "gemini-3.1-flash-lite" {
		t.Fatalf("CatalogModel = %q, want gemini-3.1-flash-lite", pm.CatalogModel)
	}
}

func TestValidateConfig_GeminiGlobalDefaultModelAutoFix(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "gemini"
	cfg.DefaultModel = "gemini-2.0-flash-lite"

	result := ValidateConfig(cfg)
	issue := findValidationIssue(result, "default_model")
	if issue == nil || issue.Severity != ValidationSeverityError || !issue.CanAutoFix {
		t.Fatalf("default_model issue = %#v, want autofixable error", issue)
	}
	if fixed := ApplyAutoFixes(cfg, result); fixed != 1 {
		t.Fatalf("ApplyAutoFixes() = %d, want 1", fixed)
	}
	if cfg.DefaultModel != "gemini-3.1-flash-lite" {
		t.Fatalf("DefaultModel = %q, want gemini-3.1-flash-lite", cfg.DefaultModel)
	}
}

func TestValidateConfig_GeminiFunctionCallingAutoFixesDefaultAndCatalog(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"gemini": {
			DefaultModel: "gemini-2.0-flash-lite",
			CatalogModel: "models/gemini-2.0-flash-lite",
		},
	})

	result := ValidateConfig(cfg)
	defaultIssue := findValidationIssue(result, "provider_models.gemini.default_model")
	if defaultIssue == nil || !defaultIssue.CanAutoFix {
		t.Fatalf("default_model issue = %#v, want autofixable issue", defaultIssue)
	}
	catalogIssue := findValidationIssue(result, "provider_models.gemini.catalog_model")
	if catalogIssue == nil || !catalogIssue.CanAutoFix {
		t.Fatalf("catalog_model issue = %#v, want autofixable issue", catalogIssue)
	}

	if fixed := ApplyAutoFixes(cfg, result); fixed != 2 {
		t.Fatalf("ApplyAutoFixes() = %d, want 2", fixed)
	}
	pm, _ := cfg.rawExplicitProviderModelConfig("gemini")
	if pm.DefaultModel != "gemini-3.1-flash-lite" {
		t.Fatalf("DefaultModel = %q, want gemini-3.1-flash-lite", pm.DefaultModel)
	}
	if pm.CatalogModel != "gemini-3.1-flash-lite" {
		t.Fatalf("CatalogModel = %q, want gemini-3.1-flash-lite", pm.CatalogModel)
	}
	if result := ValidateConfig(cfg); !result.Valid {
		t.Fatalf("ValidateConfig() after autofix = %#v, want valid", result.Issues)
	}
}

func TestValidateConfig_GeminiFunctionCallingOverrideKeyUnsupportedIsManual(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"gemini": {
			ModelOverrides: map[string]ModelOverride{
				"gemini-2.0-flash-lite": {CatalogModel: "gemini-3.5-flash"},
			},
		},
	})

	result := ValidateConfig(cfg)
	keyIssue := findValidationIssue(result, "provider_models.gemini.model_overrides.gemini-2.0-flash-lite")
	if keyIssue == nil || keyIssue.CanAutoFix {
		t.Fatalf("override key issue = %#v, want non-autofixable issue", keyIssue)
	}
	catalogIssue := findValidationIssue(result, "provider_models.gemini.model_overrides.gemini-2.0-flash-lite.catalog_model")
	if catalogIssue != nil {
		t.Fatalf("catalog_model issue = %#v, want no catalog autofix for unsupported override key", catalogIssue)
	}
	if fixed := ApplyAutoFixes(cfg, result); fixed != 0 {
		t.Fatalf("ApplyAutoFixes() = %d, want 0", fixed)
	}
	pm, _ := cfg.rawExplicitProviderModelConfig("gemini")
	if got := pm.ModelOverrides["gemini-2.0-flash-lite"].CatalogModel; got != "gemini-3.5-flash" {
		t.Fatalf("override CatalogModel = %q, want unchanged gemini-3.5-flash", got)
	}
}

func findValidationIssue(result ValidationResult, field string) *ValidationIssue {
	for i := range result.Issues {
		if result.Issues[i].Field == field {
			return &result.Issues[i]
		}
	}
	return nil
}
