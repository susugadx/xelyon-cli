package claude

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseClaude_MissingAPIKeyFailsUnlessPrintRequest(t *testing.T) {
	setClaudeDiagnosticTestEnv(t, "", "")

	report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
	requireClaudeDiagnosticCheckStatus(t, report, "auth", DiagnosticStatusFail)

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		t.Fatalf("print request should not send network traffic")
	}))
	defer server.Close()

	t.Setenv(anthropicAPIURLEnv, server.URL)
	report = Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultClaudeModel,
		CatalogModel: defaultClaudeModel,
		PrintRequest: true,
		ToolSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false for --print-request without API key: %#v", report.Checks)
	}
	if requestCount != 0 {
		t.Fatalf("requestCount = %d, want no network", requestCount)
	}
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 1 {
		t.Fatalf("RequestPreview = %#v, want one request", report.RequestPreview)
	}
	requireNoClaudeDiagnosticChecks(t, report, "auth")
}

func TestDiagnoseClaude_ModelAndCatalogResolution(t *testing.T) {
	setClaudeDiagnosticTestEnv(t, "", "claude-key")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("claude", config.ProviderModelConfig{
		DefaultModel: "corp-claude-model",
		CatalogModel: defaultClaudeModel,
	})

	report := Diagnose(context.Background(), DiagnosticOptions{Config: cfg})
	if report.Model != "corp-claude-model" || report.ModelSource != "provider_models.claude.default_model" {
		t.Fatalf("model = %q (%s), want provider model config", report.Model, report.ModelSource)
	}
	if report.CatalogModel != defaultClaudeModel || report.CatalogModelSource != "provider_models.claude.catalog_model" {
		t.Fatalf("catalog_model = %q (%s), want provider catalog config", report.CatalogModel, report.CatalogModelSource)
	}

	report = Diagnose(context.Background(), DiagnosticOptions{
		Config:       cfg,
		Model:        "explicit-claude-model",
		CatalogModel: "claude-opus-4-7",
	})
	if report.Model != "explicit-claude-model" || report.ModelSource != "--model" {
		t.Fatalf("explicit model = %q (%s), want --model", report.Model, report.ModelSource)
	}
	if report.CatalogModel != "claude-opus-4-7" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("explicit catalog = %q (%s), want --catalog-model", report.CatalogModel, report.CatalogModelSource)
	}

	fallback := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
	if fallback.Model != defaultClaudeModel || fallback.ModelSource != "built-in provider default" {
		t.Fatalf("fallback model = %q (%s), want built-in Claude default", fallback.Model, fallback.ModelSource)
	}
}

func TestDiagnoseClaude_NonClaudeCatalogModelDoesNotUseGlobalMetadata(t *testing.T) {
	setClaudeDiagnosticTestEnv(t, "", "claude-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "corp-claude-model",
		CatalogModel: "gpt-5.5",
	})
	if report.CatalogModel != "gpt-5.5" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit non-Claude catalog", report.CatalogModel, report.CatalogModelSource)
	}
	if report.ContextWindowTokens != 0 {
		t.Fatalf("ContextWindowTokens = %d, want non-Claude metadata ignored", report.ContextWindowTokens)
	}
	if report.MaxOutputTokens != 64000 {
		t.Fatalf("MaxOutputTokens = %d, want Claude runtime fallback", report.MaxOutputTokens)
	}
	requireClaudeDiagnosticCheckStatus(t, report, "catalog_model", DiagnosticStatusWarn)
	catalogPolicy := requireClaudeDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusWarn)
	if !strings.Contains(catalogPolicy.Detail, "context_window=unknown") ||
		!strings.Contains(catalogPolicy.Detail, "max_output_tokens=unknown") ||
		strings.Contains(catalogPolicy.Detail, "1050000") ||
		strings.Contains(catalogPolicy.Detail, "128000") {
		t.Fatalf("catalog_policy detail = %q, want no OpenAI metadata", catalogPolicy.Detail)
	}
}

func TestDiagnoseClaude_CatalogModelsKnownByPricingMetadata(t *testing.T) {
	setClaudeDiagnosticTestEnv(t, "", "claude-key")

	for _, catalogModel := range []string{"claude-sonnet-4-20250514", "claude-3-opus-20240229"} {
		t.Run(catalogModel, func(t *testing.T) {
			report := Diagnose(context.Background(), DiagnosticOptions{
				Config:       config.DefaultConfig(),
				Model:        "corp-claude-model",
				CatalogModel: catalogModel,
			})
			if report.CatalogModel != catalogModel || report.CatalogModelSource != "--catalog-model" {
				t.Fatalf("catalog_model = %q (%s), want explicit %s", report.CatalogModel, report.CatalogModelSource, catalogModel)
			}
			if report.ContextWindowTokens != 200000 || report.MaxOutputTokens == 0 {
				t.Fatalf("token policy = context %d max %d, want Claude metadata", report.ContextWindowTokens, report.MaxOutputTokens)
			}
			requireClaudeDiagnosticCheckStatus(t, report, "catalog_model", DiagnosticStatusOK)
			requireClaudeDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusOK)
		})
	}
}

func requireClaudeDiagnosticCheckStatus(t *testing.T, report DiagnosticReport, name string, status DiagnosticStatus) DiagnosticCheck {
	t.Helper()
	for _, check := range report.Checks {
		if check.Name == name {
			if check.Status != status {
				t.Fatalf("%s check status = %q, want %q: %#v", name, check.Status, status, check)
			}
			return check
		}
	}
	t.Fatalf("missing %s check: %#v", name, report.Checks)
	return DiagnosticCheck{}
}

func requireNoClaudeDiagnosticChecks(t *testing.T, report DiagnosticReport, names ...string) {
	t.Helper()
	for _, check := range report.Checks {
		for _, name := range names {
			if check.Name == name {
				t.Fatalf("%s check should be skipped: %#v", check.Name, report.Checks)
			}
		}
	}
}
