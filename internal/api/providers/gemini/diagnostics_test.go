package gemini

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseGemini_MissingAPIKeyFailsUnlessPrintRequest(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "")
	t.Setenv(geminiAPIURLEnv, "")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
	requireGeminiDiagnosticCheckStatus(t, report, "auth", DiagnosticStatusFail)

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		t.Fatalf("print request should not send network traffic")
	}))
	defer server.Close()

	t.Setenv(geminiAPIURLEnv, server.URL)
	report = Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultGeminiDiagnosticModel,
		CatalogModel: defaultGeminiDiagnosticModel,
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
	requireNoGeminiDiagnosticChecks(t, report, "auth")
}

func TestDiagnoseGemini_ModelAndCatalogResolution(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "gemini-key")
	t.Setenv(geminiAPIURLEnv, "")
	t.Setenv("XELYON_MODEL", "")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("gemini", config.ProviderModelConfig{
		DefaultModel: "corp-gemini-model",
		CatalogModel: defaultGeminiDiagnosticModel,
	})

	report := Diagnose(context.Background(), DiagnosticOptions{Config: cfg})
	if report.Model != "corp-gemini-model" || report.ModelSource != "provider_models.gemini.default_model" {
		t.Fatalf("model = %q (%s), want provider model config", report.Model, report.ModelSource)
	}
	if report.CatalogModel != defaultGeminiDiagnosticModel || report.CatalogModelSource != "provider_models.gemini.catalog_model" {
		t.Fatalf("catalog_model = %q (%s), want provider catalog config", report.CatalogModel, report.CatalogModelSource)
	}

	report = Diagnose(context.Background(), DiagnosticOptions{
		Config:       cfg,
		Model:        "explicit-gemini-model",
		CatalogModel: "gemini-2.5-flash",
	})
	if report.Model != "explicit-gemini-model" || report.ModelSource != "--model" {
		t.Fatalf("explicit model = %q (%s), want --model", report.Model, report.ModelSource)
	}
	if report.CatalogModel != "gemini-2.5-flash" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("explicit catalog = %q (%s), want --catalog-model", report.CatalogModel, report.CatalogModelSource)
	}

	fallback := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
	if fallback.Model != defaultGeminiDiagnosticModel || fallback.ModelSource != "built-in provider default" {
		t.Fatalf("fallback model = %q (%s), want built-in Gemini default", fallback.Model, fallback.ModelSource)
	}
}

func TestDiagnoseGemini_NonGeminiCatalogModelDoesNotUseGlobalMetadata(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "gemini-key")
	t.Setenv(geminiAPIURLEnv, "")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "corp-gemini-model",
		CatalogModel: "gpt-5.5",
	})
	if report.CatalogModel != "gpt-5.5" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit non-Gemini catalog", report.CatalogModel, report.CatalogModelSource)
	}
	if report.ContextWindowTokens != 0 {
		t.Fatalf("ContextWindowTokens = %d, want non-Gemini metadata ignored", report.ContextWindowTokens)
	}
	if report.MaxOutputTokens == 128000 {
		t.Fatalf("MaxOutputTokens = %d, should not use OpenAI max output metadata", report.MaxOutputTokens)
	}
	requireGeminiDiagnosticCheckStatus(t, report, "catalog_model", DiagnosticStatusWarn)
	catalogPolicy := requireGeminiDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusWarn)
	if !strings.Contains(catalogPolicy.Detail, "context_window=unknown") ||
		!strings.Contains(catalogPolicy.Detail, "max_output_tokens=unknown") ||
		strings.Contains(catalogPolicy.Detail, "1050000") ||
		strings.Contains(catalogPolicy.Detail, "128000") {
		t.Fatalf("catalog_policy detail = %q, want no OpenAI metadata", catalogPolicy.Detail)
	}
}

func TestDiagnoseGemini_FunctionCallingCapabilityUsesCatalogModel(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "gemini-key")
	t.Setenv(geminiAPIURLEnv, "")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "corp-lite",
		CatalogModel: "gemini-2.0-flash-lite",
	})
	if report.FunctionCallingEnabled {
		t.Fatalf("FunctionCallingEnabled = true, want false for unsupported catalog_model")
	}
	functionCalling := requireGeminiDiagnosticCheckStatus(t, report, "function_calling", DiagnosticStatusFail)
	if !strings.Contains(functionCalling.Detail, "catalog_model=gemini-2.0-flash-lite") ||
		!strings.Contains(functionCalling.Detail, "supported=false") ||
		!strings.Contains(functionCalling.Suggestion, "gemini-3.1-flash-lite") {
		t.Fatalf("function_calling check = %#v, want unsupported catalog_model guidance", functionCalling)
	}

	report = Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "corp-lite",
		CatalogModel: "models/gemini-2.0-flash-lite",
	})
	if report.CatalogModel != "gemini-2.0-flash-lite" {
		t.Fatalf("CatalogModel = %q, want canonical Gemini catalog model", report.CatalogModel)
	}
	if report.FunctionCallingEnabled {
		t.Fatalf("FunctionCallingEnabled = true, want false for unsupported canonical catalog_model")
	}
	functionCalling = requireGeminiDiagnosticCheckStatus(t, report, "function_calling", DiagnosticStatusFail)
	if !strings.Contains(functionCalling.Detail, "catalog_model=gemini-2.0-flash-lite") ||
		!strings.Contains(functionCalling.Detail, "supported=false") ||
		!strings.Contains(functionCalling.Suggestion, "gemini-3.1-flash-lite") {
		t.Fatalf("function_calling check = %#v, want unsupported canonical catalog_model guidance", functionCalling)
	}
	requireGeminiDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusOK)

	report = Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "corp-flash",
		CatalogModel: "gemini-3.5-flash",
	})
	if !report.FunctionCallingEnabled {
		t.Fatalf("FunctionCallingEnabled = false, want true for supported catalog_model")
	}
	functionCalling = requireGeminiDiagnosticCheckStatus(t, report, "function_calling", DiagnosticStatusOK)
	if !strings.Contains(functionCalling.Detail, "catalog_model=gemini-3.5-flash") ||
		!strings.Contains(functionCalling.Detail, "supported=true") {
		t.Fatalf("function_calling check = %#v, want supported catalog_model detail", functionCalling)
	}

	report = Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gemini-2.0-flash-lite",
		CatalogModel: "gemini-3.5-flash",
	})
	if report.FunctionCallingEnabled {
		t.Fatalf("FunctionCallingEnabled = true, want false for unsupported request model")
	}
	functionCalling = requireGeminiDiagnosticCheckStatus(t, report, "function_calling", DiagnosticStatusFail)
	for _, fragment := range []string{
		"request_model=gemini-2.0-flash-lite",
		"catalog_model=gemini-3.5-flash",
		"policy_model=gemini-2.0-flash-lite",
		"supported=false",
	} {
		if !strings.Contains(functionCalling.Detail, fragment) {
			t.Fatalf("function_calling detail = %q, want %q", functionCalling.Detail, fragment)
		}
	}

	report = Diagnose(context.Background(), DiagnosticOptions{
		Config: config.DefaultConfig(),
		Model:  "corp-unknown",
	})
	if !report.FunctionCallingEnabled {
		t.Fatalf("FunctionCallingEnabled = false, want optimistic true for unknown alias")
	}
	functionCalling = requireGeminiDiagnosticCheckStatus(t, report, "function_calling", DiagnosticStatusWarn)
	if !strings.Contains(functionCalling.Detail, "catalog_model=corp-unknown") ||
		!strings.Contains(functionCalling.Suggestion, "--catalog-model") {
		t.Fatalf("function_calling check = %#v, want catalog_model guidance for unknown alias", functionCalling)
	}
}

func TestDiagnoseGemini_ModelLifecycleWarnings(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "gemini-key")
	t.Setenv(geminiAPIURLEnv, "")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gemini-3.1-pro",
		CatalogModel: "gemini-3.1-pro",
	})
	lifecycle := requireGeminiDiagnosticCheckStatus(t, report, "model_lifecycle", DiagnosticStatusWarn)
	if !strings.Contains(lifecycle.Detail, "stage=active") ||
		!strings.Contains(lifecycle.Detail, "picker=hidden") ||
		!strings.Contains(lifecycle.Detail, "replacement=gemini-3.1-pro-preview-customtools") {
		t.Fatalf("model_lifecycle detail = %q, want active hidden replacement detail", lifecycle.Detail)
	}

	report = Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gemini-3-pro-preview",
		CatalogModel: "gemini-3-pro-preview",
	})
	lifecycle = requireGeminiDiagnosticCheckStatus(t, report, "model_lifecycle", DiagnosticStatusWarn)
	if !strings.Contains(lifecycle.Message, "shut down") ||
		!strings.Contains(lifecycle.Detail, "stage=shutdown") ||
		!strings.Contains(lifecycle.Detail, "shutdown_date=2026-03-09") {
		t.Fatalf("model_lifecycle = %#v, want shutdown warning", lifecycle)
	}

	report = Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gemini-3-pro-preview",
		CatalogModel: "gemini-3.5-flash",
	})
	lifecycle = requireGeminiDiagnosticCheckStatus(t, report, "model_lifecycle", DiagnosticStatusWarn)
	if !strings.Contains(lifecycle.Message, "request model has been shut down") ||
		!strings.Contains(lifecycle.Detail, "request_model{model=gemini-3-pro-preview") ||
		!strings.Contains(lifecycle.Detail, "stage=shutdown") ||
		!strings.Contains(lifecycle.Detail, "shutdown_date=2026-03-09") ||
		strings.Contains(lifecycle.Detail, "catalog_model{") {
		t.Fatalf("model_lifecycle = %#v, want request model shutdown warning without catalog warning", lifecycle)
	}

	report = Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gemini-1.5-pro-001",
		PrintRequest: true,
	})
	lifecycle = requireGeminiDiagnosticCheckStatus(t, report, "model_lifecycle", DiagnosticStatusWarn)
	if !strings.Contains(lifecycle.Message, "shut down") ||
		!strings.Contains(lifecycle.Detail, "model=gemini-1.5-pro-001") ||
		!strings.Contains(lifecycle.Detail, "stage=shutdown") ||
		!strings.Contains(lifecycle.Detail, "shutdown_date=2025-09-29") {
		t.Fatalf("model_lifecycle = %#v, want suffixed Gemini 1.5 shutdown warning", lifecycle)
	}
}

func TestDiagnoseGemini_EndpointCheckMatchesSelectedRoutes(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "gemini-key")
	t.Setenv("XELYON_MODEL", "")

	model := defaultGeminiDiagnosticModel
	streamURL := "https://proxy.example/v1beta/models/" + model + ":streamGenerateContent?alt=sse"
	t.Setenv(geminiAPIURLEnv, streamURL)

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        model,
		CatalogModel: model,
		PrintRequest: true,
		TextSmoke:    true,
	})
	endpoint := requireGeminiDiagnosticCheckStatus(t, report, "endpoint", DiagnosticStatusOK)
	if !strings.Contains(endpoint.Detail, "stream_url="+streamURL) {
		t.Fatalf("endpoint detail = %q, want stream URL detail", endpoint.Detail)
	}

	streamURLWithoutSSE := "https://proxy.example/v1beta/models/" + model + ":streamGenerateContent"
	t.Setenv(geminiAPIURLEnv, streamURLWithoutSSE)
	report = Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        model,
		CatalogModel: model,
		PrintRequest: true,
		TextSmoke:    true,
	})
	endpoint = requireGeminiDiagnosticCheckStatus(t, report, "endpoint", DiagnosticStatusWarn)
	if !strings.Contains(endpoint.Detail, "alt=sse for streamGenerateContent text/tool/image") {
		t.Fatalf("endpoint detail = %q, want missing alt=sse warning", endpoint.Detail)
	}

	t.Setenv(geminiAPIURLEnv, streamURL)
	report = Diagnose(context.Background(), DiagnosticOptions{
		Config:         config.DefaultConfig(),
		Model:          model,
		CatalogModel:   model,
		PrintRequest:   true,
		WebSearchSmoke: true,
	})
	endpoint = requireGeminiDiagnosticCheckStatus(t, report, "endpoint", DiagnosticStatusWarn)
	if !strings.Contains(endpoint.Detail, "generateContent for native web search") {
		t.Fatalf("endpoint detail = %q, want selected web search route warning", endpoint.Detail)
	}
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 1 {
		t.Fatalf("RequestPreview = %#v, want one web search request", report.RequestPreview)
	}
	preview := report.RequestPreview.Requests[0]
	if preview.Route != DiagnosticRouteGenerateContent || preview.URL != streamURL {
		t.Fatalf("preview route/url = %s %s, want generateContent route using configured URL", preview.Route, preview.URL)
	}

	generateURL := "https://proxy.example/v1beta/models/" + model + ":generateContent"
	t.Setenv(geminiAPIURLEnv, generateURL)
	report = Diagnose(context.Background(), DiagnosticOptions{
		Config:         config.DefaultConfig(),
		Model:          model,
		CatalogModel:   model,
		PrintRequest:   true,
		WebSearchSmoke: true,
	})
	endpoint = requireGeminiDiagnosticCheckStatus(t, report, "endpoint", DiagnosticStatusOK)
	if !strings.Contains(endpoint.Detail, "generate_url="+generateURL) {
		t.Fatalf("endpoint detail = %q, want generate URL detail", endpoint.Detail)
	}

	proxyURL := "https://proxy.example/gemini"
	t.Setenv(geminiAPIURLEnv, proxyURL)
	report = Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        model,
		CatalogModel: model,
		PrintRequest: true,
		TextSmoke:    true,
	})
	endpoint = requireGeminiDiagnosticCheckStatus(t, report, "endpoint", DiagnosticStatusWarn)
	if !strings.Contains(endpoint.Detail, "streamGenerateContent?alt=sse for text/tool/image") {
		t.Fatalf("endpoint detail = %q, want selected stream route warning", endpoint.Detail)
	}
}

func TestDiagnoseGemini_InvalidEndpointFails(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "gemini-key")
	t.Setenv(geminiAPIURLEnv, "http://")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultGeminiDiagnosticModel,
		CatalogModel: defaultGeminiDiagnosticModel,
	})
	requireGeminiDiagnosticCheckStatus(t, report, "endpoint", DiagnosticStatusFail)
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want invalid endpoint failure")
	}
}

func requireGeminiDiagnosticCheckStatus(t *testing.T, report DiagnosticReport, name string, status DiagnosticStatus) DiagnosticCheck {
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

func requireNoGeminiDiagnosticChecks(t *testing.T, report DiagnosticReport, names ...string) {
	t.Helper()
	for _, check := range report.Checks {
		for _, name := range names {
			if check.Name == name {
				t.Fatalf("%s check should be skipped: %#v", check.Name, report.Checks)
			}
		}
	}
}
