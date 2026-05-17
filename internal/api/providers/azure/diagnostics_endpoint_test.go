package azure

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnose_WarnsForAPIVersionQueryAndCatalogFallback(t *testing.T) {
	t.Setenv(baseURLEnv, "https://example.openai.azure.com/openai/v1?api-version=2025-04-01-preview")
	t.Setenv(apiKeyEnv, "azure-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:     config.DefaultConfig(),
		Deployment: "corp-gpt55-deployment",
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if !hasDiagnosticCheck(report, "api_version", DiagnosticStatusWarn) {
		t.Fatalf("missing api-version warning: %#v", report.Checks)
	}
	if !hasDiagnosticCheck(report, "catalog_model", DiagnosticStatusWarn) {
		t.Fatalf("missing catalog_model fallback warning: %#v", report.Checks)
	}
	if report.NormalizedBaseURL != "https://example.openai.azure.com/openai/v1" {
		t.Fatalf("NormalizedBaseURL = %q, want v1 URL without query", report.NormalizedBaseURL)
	}
}

func TestDiagnose_FailsForDeploymentScopedBaseURL(t *testing.T) {
	t.Setenv(baseURLEnv, "https://example.openai.azure.com/openai/deployments/corp-gpt55")
	t.Setenv(apiKeyEnv, "azure-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55",
		CatalogModel: "gpt-5.5",
	})

	if !hasDiagnosticCheck(report, "base_url", DiagnosticStatusFail) {
		t.Fatalf("missing deployment URL failure: %#v", report.Checks)
	}
}

func TestDiagnose_FailsForPublicOpenAIBaseURL(t *testing.T) {
	t.Setenv(baseURLEnv, "https://api.openai.com/v1")
	t.Setenv(apiKeyEnv, "azure-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55",
		CatalogModel: "gpt-5.5",
	})

	if !hasDiagnosticCheck(report, "base_url", DiagnosticStatusFail) {
		t.Fatalf("missing public OpenAI base URL failure: %#v", report.Checks)
	}
}
