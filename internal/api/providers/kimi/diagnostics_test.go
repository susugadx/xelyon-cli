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
		{"image_input", DiagnosticStatusOK},
		{"unsupported_features", DiagnosticStatusInfo},
		{"prompt_cache_key", DiagnosticStatusOK},
	} {
		if !hasKimiDiagnosticCheck(report, want.name, want.status) {
			t.Fatalf("missing %s/%s check: %#v", want.name, want.status, report.Checks)
		}
	}
}
