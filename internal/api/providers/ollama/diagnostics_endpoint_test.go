package ollama

import (
	"context"
	"net/http"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseOllama_InstalledModelLatestMatch(t *testing.T) {
	server := newOllamaDiagnosticServer(t, []string{"llama3:latest"}, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected chat request: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	t.Setenv(ollamaBaseURLEnv, server.URL)

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "llama3",
		CatalogModel: "llama3:8b",
	})
	check, ok := ollamaDiagnosticCheckByName(report, "installed_model")
	if !ok || check.Status != DiagnosticStatusOK {
		t.Fatalf("installed_model check = %#v, %v; want ok for :latest equivalence", check, ok)
	}
}

func TestDiagnoseOllama_MissingInstalledModelFails(t *testing.T) {
	server := newOllamaDiagnosticServer(t, []string{"llama3:8b"}, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected chat request: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	t.Setenv(ollamaBaseURLEnv, server.URL)

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "qwen2.5-coder:7b",
		CatalogModel: "qwen2.5-coder:7b",
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want missing installed model failure: %#v", report.Checks)
	}
	check, ok := ollamaDiagnosticCheckByName(report, "installed_model")
	if !ok || check.Status != DiagnosticStatusFail {
		t.Fatalf("installed_model check = %#v, %v; want fail", check, ok)
	}
}

func TestDiagnoseOllama_EndpointOverrideToAPIPathFailsAndSkipsSmoke(t *testing.T) {
	server := newOllamaDiagnosticServer(t, []string{"qwen2.5-coder:7b"}, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected chat request: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	t.Setenv(ollamaBaseURLEnv, server.URL+ollamaChatEndpointPath)

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "qwen2.5-coder:7b",
		CatalogModel: "qwen2.5-coder:7b",
		RunSmoke:     true,
		TextSmoke:    true,
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want endpoint path failure: %#v", report.Checks)
	}
	check, ok := ollamaDiagnosticCheckByName(report, "endpoint")
	if !ok || check.Status != DiagnosticStatusFail {
		t.Fatalf("endpoint check = %#v, %v; want fail for endpoint path override", check, ok)
	}
	smoke, ok := ollamaDiagnosticCheckByName(report, "smoke")
	if !ok || smoke.Status != DiagnosticStatusWarn {
		t.Fatalf("smoke check = %#v, %v; want skipped smoke warning", smoke, ok)
	}
	if report.Smoke != nil {
		t.Fatalf("Smoke = %#v, want nil when prerequisite endpoint check fails", report.Smoke)
	}
}
