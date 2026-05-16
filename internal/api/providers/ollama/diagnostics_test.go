package ollama

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseOllama_ModelCatalogPolicyRouteAndFunctionCalling(t *testing.T) {
	server := newOllamaDiagnosticServer(t, []string{"qwen2.5-coder:7b"}, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected chat request: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	t.Setenv(ollamaBaseURLEnv, server.URL)
	t.Setenv("OLLAMA_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "qwen2.5-coder:7b",
		CatalogModel: "qwen2.5-coder:7b",
	})

	if report.Provider != "ollama" {
		t.Fatalf("Provider = %q, want ollama", report.Provider)
	}
	if report.APIURL != server.URL {
		t.Fatalf("APIURL = %q, want fake server URL", report.APIURL)
	}
	if report.Model != "qwen2.5-coder:7b" || report.ModelSource != "--model" {
		t.Fatalf("model = %q (%s), want explicit model", report.Model, report.ModelSource)
	}
	if report.CatalogModel != "qwen2.5-coder:7b" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit catalog model", report.CatalogModel, report.CatalogModelSource)
	}
	if report.Route != DiagnosticRouteOllamaChat {
		t.Fatalf("Route = %q, want ollama_chat", report.Route)
	}
	if report.MaxOutputTokens != 4096 {
		t.Fatalf("MaxOutputTokens = %d, want Ollama provider max output default", report.MaxOutputTokens)
	}
	if report.ContextWindowTokens != 32768 {
		t.Fatalf("ContextWindowTokens = %d, want qwen context window", report.ContextWindowTokens)
	}
	if !report.FunctionCallingEnabled {
		t.Fatal("FunctionCallingEnabled = false, want true")
	}
	for _, name := range []string{"auth", "endpoint", "provider_registration", "model", "installed_model", "catalog_model", "route", "catalog_policy", "function_calling"} {
		check, ok := ollamaDiagnosticCheckByName(report, name)
		if !ok || check.Status != DiagnosticStatusOK {
			t.Fatalf("%s check = %#v, %v; want ok", name, check, ok)
		}
	}
	catalogPolicy, _ := ollamaDiagnosticCheckByName(report, "catalog_policy")
	if !strings.Contains(catalogPolicy.Detail, "context_window=32768") ||
		!strings.Contains(catalogPolicy.Detail, "max_output_tokens=4096") ||
		!strings.Contains(catalogPolicy.Detail, "pricing=input $0.00/M") {
		t.Fatalf("catalog_policy detail = %q, want Ollama policy detail", catalogPolicy.Detail)
	}
}

func TestDiagnoseOllama_NonOllamaCatalogModelDoesNotUseGlobalMetadata(t *testing.T) {
	server := newOllamaDiagnosticServer(t, []string{"corp-ollama-model"}, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected chat request: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	t.Setenv(ollamaBaseURLEnv, server.URL)
	t.Setenv("OLLAMA_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "corp-ollama-model",
		CatalogModel: "gpt-5.5",
	})

	if report.CatalogModel != "gpt-5.5" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit value in report", report.CatalogModel, report.CatalogModelSource)
	}
	if report.ContextWindowTokens != 0 {
		t.Fatalf("ContextWindowTokens = %d, want non-Ollama catalog metadata ignored", report.ContextWindowTokens)
	}
	if report.MaxOutputTokens == 128000 {
		t.Fatalf("MaxOutputTokens = %d, want OpenAI catalog max output ignored", report.MaxOutputTokens)
	}
	catalogModel, ok := ollamaDiagnosticCheckByName(report, "catalog_model")
	if !ok || catalogModel.Status != DiagnosticStatusWarn {
		t.Fatalf("catalog_model check = %#v, %v; want warn", catalogModel, ok)
	}
	catalogPolicy, ok := ollamaDiagnosticCheckByName(report, "catalog_policy")
	if !ok || catalogPolicy.Status != DiagnosticStatusWarn {
		t.Fatalf("catalog_policy check = %#v, %v; want warn", catalogPolicy, ok)
	}
	if !strings.Contains(catalogPolicy.Detail, "context_window=unknown") ||
		!strings.Contains(catalogPolicy.Detail, "max_output_tokens=4096") ||
		strings.Contains(catalogPolicy.Detail, "1050000") ||
		strings.Contains(catalogPolicy.Detail, "128000") {
		t.Fatalf("catalog_policy detail = %q, want provider fallback without OpenAI token metadata", catalogPolicy.Detail)
	}
}

func TestDiagnoseOllama_PrintRequestDoesNotSendNetwork(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Fatalf("unexpected network request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	t.Setenv(ollamaBaseURLEnv, server.URL)
	t.Setenv("OLLAMA_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "qwen2.5-coder:7b",
		CatalogModel: "qwen2.5-coder:7b",
		ToolSmoke:    true,
		PrintRequest: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want print request to pass: %#v", report.Checks)
	}
	if report.Smoke != nil {
		t.Fatalf("Smoke = %#v, want nil for print-request", report.Smoke)
	}
	if requests != 0 {
		t.Fatalf("network requests = %d, want 0", requests)
	}
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 1 {
		t.Fatalf("RequestPreview = %#v, want one tool request", report.RequestPreview)
	}
	preview := report.RequestPreview.Requests[0]
	if preview.Name != "tool" || !preview.ToolPayload || preview.URL != server.URL+"/api/chat" {
		t.Fatalf("preview = %#v, want tool request to Ollama chat endpoint", preview)
	}
	if preview.Headers["Content-Type"] != "application/json" {
		t.Fatalf("Content-Type preview = %q, want application/json", preview.Headers["Content-Type"])
	}
	body := decodeOllamaDiagnosticPreviewBody(t, preview.Body)
	if body.Model != "qwen2.5-coder:7b" || body.Options == nil || body.Options.NumPredict != 64 || len(body.Tools) != 1 || body.ToolChoice != ollamaDiagnosticSmokeToolName {
		t.Fatalf("preview body = %#v, want forced diagnostic tool body", body)
	}
}

func TestDiagnoseOllama_PrintRequestSkipsDisabledToolPreview(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Fatalf("unexpected network request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	t.Setenv(ollamaBaseURLEnv, server.URL)
	t.Setenv("OLLAMA_FUNCTION_CALLING", "0")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "qwen2.5-coder:7b",
		CatalogModel: "qwen2.5-coder:7b",
		ToolSmoke:    true,
		PrintRequest: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want disabled tool preview skip to pass: %#v", report.Checks)
	}
	if requests != 0 {
		t.Fatalf("network requests = %d, want 0", requests)
	}
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 2 {
		t.Fatalf("RequestPreview = %#v, want text fallback plus skipped tool request", report.RequestPreview)
	}
	text := report.RequestPreview.Requests[0]
	if text.Name != "text" || text.Skipped || text.ToolPayload {
		t.Fatalf("text preview = %#v, want runnable text fallback", text)
	}
	tool := report.RequestPreview.Requests[1]
	if tool.Name != "tool" || !tool.Skipped || !tool.ToolPayload || !strings.Contains(tool.SkipReason, "OLLAMA_FUNCTION_CALLING=0") {
		t.Fatalf("tool preview = %#v, want skipped disabled tool request", tool)
	}
}
