package kimi

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnose_FailsForMissingKeyAndInvalidAPIURL(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "")
	t.Setenv(kimiAPIURLEnv, "://bad")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want true: %#v", report.Checks)
	}
	if !hasKimiDiagnosticCheck(report, "auth", DiagnosticStatusFail) {
		t.Fatalf("missing auth failure: %#v", report.Checks)
	}
	if !hasKimiDiagnosticCheck(report, "api_url", DiagnosticStatusFail) {
		t.Fatalf("missing api_url failure: %#v", report.Checks)
	}
}

func TestDiagnose_ReportsRegistrationModelUnsupportedAndPromptCacheKey(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "moonshot-key")
	t.Setenv(kimiAPIURLEnv, "")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if report.Model != defaultKimiModel {
		t.Fatalf("Model = %q, want %q", report.Model, defaultKimiModel)
	}
	if !report.PromptCacheKeyPresent {
		t.Fatal("PromptCacheKeyPresent = false, want true")
	}
	for _, unsupported := range report.UnsupportedFeatures {
		if unsupported == "image input" {
			t.Fatalf("UnsupportedFeatures = %v, want image input removed", report.UnsupportedFeatures)
		}
		if unsupported == "built-in web_search" {
			t.Fatalf("UnsupportedFeatures = %v, want built-in web_search removed", report.UnsupportedFeatures)
		}
	}
	for _, want := range []struct {
		name   string
		status DiagnosticStatus
	}{
		{"provider_registration", DiagnosticStatusOK},
		{"model", DiagnosticStatusOK},
		{"catalog_model", DiagnosticStatusOK},
		{"route", DiagnosticStatusOK},
		{"catalog_policy", DiagnosticStatusOK},
		{"image_input", DiagnosticStatusOK},
		{"unsupported_features", DiagnosticStatusInfo},
		{"prompt_cache_key", DiagnosticStatusOK},
	} {
		if !hasKimiDiagnosticCheck(report, want.name, want.status) {
			t.Fatalf("missing %s/%s check: %#v", want.name, want.status, report.Checks)
		}
	}
}

func TestDiagnose_CatalogModelOptionDrivesTokenAndPricingPolicy(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "moonshot-key")
	t.Setenv(kimiAPIURLEnv, "")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "corp-kimi-model",
		CatalogModel: "kimi-k2.6",
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if report.CatalogModel != "kimi-k2.6" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("CatalogModel = %q (%s), want explicit kimi-k2.6", report.CatalogModel, report.CatalogModelSource)
	}
	if report.MaxOutputTokens != 32768 || report.ContextWindowTokens != 256000 {
		t.Fatalf("token policy = max %d context %d, want Kimi catalog values", report.MaxOutputTokens, report.ContextWindowTokens)
	}
	if !hasKimiDiagnosticCheck(report, "catalog_policy", DiagnosticStatusOK) {
		t.Fatalf("missing catalog_policy OK check: %#v", report.Checks)
	}
}

func TestDiagnose_NonKimiCatalogModelDoesNotUseGlobalMetadata(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "moonshot-key")
	t.Setenv(kimiAPIURLEnv, "")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "corp-kimi-model",
		CatalogModel: "gpt-5.5",
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want warn-only non-Kimi catalog: %#v", report.Checks)
	}
	if report.ContextWindowTokens != 0 || report.MaxOutputTokens != 0 {
		t.Fatalf("Kimi policy used non-Kimi metadata: context=%d max=%d", report.ContextWindowTokens, report.MaxOutputTokens)
	}
	if !hasKimiDiagnosticCheck(report, "catalog_model", DiagnosticStatusWarn) {
		t.Fatalf("missing catalog_model warning: %#v", report.Checks)
	}
	if !hasKimiDiagnosticCheck(report, "catalog_policy", DiagnosticStatusWarn) {
		t.Fatalf("missing catalog_policy warning: %#v", report.Checks)
	}
	catalogPolicy, ok := kimiDiagnosticCheckByName(report, "catalog_policy")
	if !ok {
		t.Fatalf("missing catalog_policy check: %#v", report.Checks)
	}
	if got, want := catalogPolicy.Detail, "catalog_model=gpt-5.5, context_window=unknown, max_output_tokens=unknown, pricing=unavailable"; got != want {
		t.Fatalf("catalog_policy detail = %q, want %q", got, want)
	}
}
