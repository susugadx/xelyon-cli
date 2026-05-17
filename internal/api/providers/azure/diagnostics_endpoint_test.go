package azure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnose_WarnsForAPIVersionQueryAndCatalogFallback(t *testing.T) {
	report := diagnoseEndpointWithAPIKey(t, "https://example.openai.azure.com/openai/v1?api-version=2025-04-01-preview", DiagnosticOptions{
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
	report := diagnoseEndpointWithAPIKey(t, "https://example.openai.azure.com/openai/deployments/corp-gpt55", DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55",
		CatalogModel: "gpt-5.5",
	})

	if !hasDiagnosticCheck(report, "base_url", DiagnosticStatusFail) {
		t.Fatalf("missing deployment URL failure: %#v", report.Checks)
	}
}

func TestDiagnose_FailsForPublicOpenAIBaseURL(t *testing.T) {
	report := diagnoseEndpointWithAPIKey(t, "https://api.openai.com/v1", DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55",
		CatalogModel: "gpt-5.5",
	})

	if !hasDiagnosticCheck(report, "base_url", DiagnosticStatusFail) {
		t.Fatalf("missing public OpenAI base URL failure: %#v", report.Checks)
	}
}

func TestDiagnose_NormalizesResourceBaseURLWithoutPathWarning(t *testing.T) {
	for _, rawBaseURL := range []string{
		"https://example.openai.azure.com",
		"https://example.openai.azure.com/openai",
	} {
		t.Run(rawBaseURL, func(t *testing.T) {
			report := diagnoseEndpointWithAPIKey(t, rawBaseURL, DiagnosticOptions{
				Config:       config.DefaultConfig(),
				Deployment:   "corp-gpt55",
				CatalogModel: "gpt-5.5",
			})

			if report.HasFailures() {
				t.Fatalf("HasFailures() = true, want false for normalizable resource base URL: %#v", report.Checks)
			}
			if report.NormalizedBaseURL != "https://example.openai.azure.com/openai/v1" {
				t.Fatalf("NormalizedBaseURL = %q, want v1 base URL", report.NormalizedBaseURL)
			}
			if _, ok := diagnosticCheckByName(report, "base_url_path"); ok {
				t.Fatalf("base_url_path warning should be omitted for normalized resource base URL: %#v", report.Checks)
			}
			if !hasDiagnosticCheck(report, "base_url", DiagnosticStatusOK) {
				t.Fatalf("missing base_url OK check: %#v", report.Checks)
			}
		})
	}
}

func TestDiagnose_WarnsForIntentionalProxyBaseURLPath(t *testing.T) {
	report := diagnoseEndpointWithAPIKey(t, "https://example.openai.azure.com/proxy/azure", DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55",
		CatalogModel: "gpt-5.5",
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false for warn-only proxy path: %#v", report.Checks)
	}
	if report.NormalizedBaseURL != "https://example.openai.azure.com/proxy/azure" {
		t.Fatalf("NormalizedBaseURL = %q, want proxy base URL preserved", report.NormalizedBaseURL)
	}
	check, ok := diagnosticCheckByName(report, "base_url_path")
	if !ok {
		t.Fatalf("missing base_url_path warning: %#v", report.Checks)
	}
	if check.Status != DiagnosticStatusWarn {
		t.Fatalf("base_url_path status = %s, want warn: %#v", check.Status, check)
	}
	if !hasDiagnosticCheck(report, "base_url", DiagnosticStatusOK) {
		t.Fatalf("missing base_url OK check for proxy path: %#v", report.Checks)
	}
}

func TestDiagnose_PrintRequestUsesProxyBaseURLWithoutLiveRequestOrAuthCommand(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL+"/proxy/azure")
	t.Setenv(apiKeyEnv, "")
	t.Setenv(authTokenEnv, "")
	t.Setenv(authTokenCommandEnv, "definitely-not-executed-token-command")
	t.Setenv(authTokenCommandTimeoutEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55-pro-deployment",
		CatalogModel: "gpt-5.5-pro",
		PrintRequest: true,
		TextSmoke:    true,
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false for request preview without auth command execution: %#v", report.Checks)
	}
	if requestCount != 0 {
		t.Fatalf("requestCount = %d, want no live request for --print-request", requestCount)
	}
	if !hasDiagnosticCheck(report, "base_url_path", DiagnosticStatusWarn) {
		t.Fatalf("missing proxy path warning: %#v", report.Checks)
	}
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 1 {
		t.Fatalf("RequestPreview = %#v, want one text request preview", report.RequestPreview)
	}
	request := report.RequestPreview.Requests[0]
	wantURL := server.URL + "/proxy/azure/responses"
	if request.Name != "text" || request.URL != wantURL || request.Method != "POST" {
		t.Fatalf("request preview = %#v, want text POST %s", request, wantURL)
	}
	if request.Headers["Authorization"] != "Bearer <redacted>" {
		t.Fatalf("headers = %#v, want redacted Authorization from token command auth mode", request.Headers)
	}
}

func TestDiagnose_SmokeUsesProxyBaseURLPath(t *testing.T) {
	var received struct {
		Path string
		Body map[string]any
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&received.Body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_proxy","output_text":"xelyon azure doctor ok","usage":{"input_tokens":10,"output_tokens":4}}`))
	}))
	defer server.Close()

	report := diagnoseEndpointWithAPIKey(t, server.URL+"/proxy/azure", DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55-pro-deployment",
		CatalogModel: "gpt-5.5-pro",
		RunSmoke:     true,
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false for proxy path smoke: %#v", report.Checks)
	}
	if !hasDiagnosticCheck(report, "base_url_path", DiagnosticStatusWarn) {
		t.Fatalf("missing proxy path warning: %#v", report.Checks)
	}
	if received.Path != "/proxy/azure/responses" {
		t.Fatalf("path = %q, want /proxy/azure/responses", received.Path)
	}
	if received.Body["model"] != "corp-gpt55-pro-deployment" {
		t.Fatalf("model = %#v, want deployment", received.Body["model"])
	}
	if report.Smoke == nil || !report.Smoke.Ran || report.Smoke.ResponseID != "resp_proxy" {
		t.Fatalf("Smoke = %#v, want proxy smoke response", report.Smoke)
	}
}

func diagnoseEndpointWithAPIKey(t *testing.T, baseURL string, options DiagnosticOptions) DiagnosticReport {
	t.Helper()
	t.Setenv(baseURLEnv, baseURL)
	t.Setenv(apiKeyEnv, "azure-key")
	t.Setenv(authTokenEnv, "")
	t.Setenv(authTokenCommandEnv, "")
	t.Setenv(authTokenCommandTimeoutEnv, "")
	return Diagnose(context.Background(), options)
}
