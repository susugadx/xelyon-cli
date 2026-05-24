package ollama

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func TestDiagnoseOllama_CapabilitiesDoNotListInstalledModels(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Fatalf("unexpected network request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	t.Setenv(ollamaBaseURLEnv, server.URL)
	t.Setenv("OLLAMA_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "qwen2.5-coder:7b",
		CatalogModel:         "qwen2.5-coder:7b",
		Capabilities:         true,
		RequiredCapabilities: []string{"function_calling"},
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want capability-only diagnostics without endpoint lookup to pass: %#v", report.Checks)
	}
	if requests != 0 {
		t.Fatalf("network requests = %d, want 0", requests)
	}
	if _, ok := ollamaDiagnosticCheckByName(report, "endpoint"); ok {
		t.Fatalf("endpoint check was added for capability-only report: %#v", report.Checks)
	}
	if _, ok := ollamaDiagnosticCheckByName(report, "installed_model"); ok {
		t.Fatalf("installed_model check was added for capability-only report: %#v", report.Checks)
	}
	if report.Capabilities == nil || !report.Capabilities.FunctionCalling {
		t.Fatalf("Capabilities = %+v, want function_calling enabled", report.Capabilities)
	}
	if report.Capabilities.LocalModelAvailable || report.Capabilities.LocalModelAvailableKnown {
		t.Fatalf("local_model_available = %t known=%t, want unknown without /api/tags discovery", report.Capabilities.LocalModelAvailable, report.Capabilities.LocalModelAvailableKnown)
	}
	capabilityCheck, ok := ollamaDiagnosticCheckByName(report, "capabilities")
	if !ok || !strings.Contains(capabilityCheck.Detail, "local_model_available=unknown") {
		t.Fatalf("capabilities check = %#v, %v; want local_model_available=unknown detail", capabilityCheck, ok)
	}
	check, ok := ollamaDiagnosticCheckByName(report, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusOK {
		t.Fatalf("required_capability check = %#v, %v; want ok", check, ok)
	}
}

func TestDiagnoseOllama_CapabilitiesPrintRequestLeavesLocalModelAvailabilityUnknown(t *testing.T) {
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
		Capabilities: true,
		PrintRequest: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want preview capabilities to pass without /api/tags lookup: %#v", report.Checks)
	}
	if requests != 0 {
		t.Fatalf("network requests = %d, want 0", requests)
	}
	if check, ok := ollamaDiagnosticCheckByName(report, "installed_model"); !ok || check.Status != DiagnosticStatusOK || !strings.Contains(check.Detail, "--print-request") {
		t.Fatalf("installed_model check = %#v, %v; want preview skip", check, ok)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if report.Capabilities.LocalModelAvailable || report.Capabilities.LocalModelAvailableKnown {
		t.Fatalf("local_model_available = %t known=%t, want unknown when --print-request skips /api/tags", report.Capabilities.LocalModelAvailable, report.Capabilities.LocalModelAvailableKnown)
	}
	capabilityCheck, ok := ollamaDiagnosticCheckByName(report, "capabilities")
	if !ok || !strings.Contains(capabilityCheck.Detail, "local_model_available=unknown") {
		t.Fatalf("capabilities check = %#v, %v; want local_model_available=unknown detail", capabilityCheck, ok)
	}
}

func TestDiagnoseOllama_RequireLocalModelAvailableListsInstalledModels(t *testing.T) {
	tests := []struct {
		name                string
		installedModels     []string
		wantFailures        bool
		wantInstalledStatus DiagnosticStatus
		wantRequiredStatus  DiagnosticStatus
		wantRequiredDetail  string
	}{
		{
			name:                "installed",
			installedModels:     []string{"qwen2.5-coder:7b"},
			wantInstalledStatus: DiagnosticStatusOK,
			wantRequiredStatus:  DiagnosticStatusOK,
			wantRequiredDetail:  "local_model_available=ok",
		},
		{
			name:                "missing",
			installedModels:     []string{"llama3.2:latest"},
			wantFailures:        true,
			wantInstalledStatus: DiagnosticStatusFail,
			wantRequiredStatus:  DiagnosticStatusFail,
			wantRequiredDetail:  "local_model_available=missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newOllamaDiagnosticServer(t, tt.installedModels, func(w http.ResponseWriter, r *http.Request) {
				t.Fatalf("unexpected chat request: %s %s", r.Method, r.URL.Path)
			})
			defer server.Close()

			t.Setenv(ollamaBaseURLEnv, server.URL)
			t.Setenv("OLLAMA_FUNCTION_CALLING", "1")

			report := Diagnose(context.Background(), DiagnosticOptions{
				Config:               config.DefaultConfig(),
				Model:                "qwen2.5-coder:7b",
				CatalogModel:         "qwen2.5-coder:7b",
				RequiredCapabilities: []string{providerdiag.RequiredCapabilityLocalModelAvailable},
			})
			if got := report.HasFailures(); got != tt.wantFailures {
				t.Fatalf("HasFailures() = %t, want %t; checks=%#v", got, tt.wantFailures, report.Checks)
			}
			if report.Capabilities != nil {
				t.Fatalf("Capabilities = %#v, want nil without --capabilities", report.Capabilities)
			}
			if check, ok := ollamaDiagnosticCheckByName(report, "endpoint"); !ok || check.Status != DiagnosticStatusOK {
				t.Fatalf("endpoint check = %#v, %v; want ok", check, ok)
			}
			if check, ok := ollamaDiagnosticCheckByName(report, "installed_model"); !ok || check.Status != tt.wantInstalledStatus {
				t.Fatalf("installed_model check = %#v, %v; want %s", check, ok, tt.wantInstalledStatus)
			}
			check, ok := ollamaDiagnosticCheckByName(report, providerdiag.RequiredCapabilityCheckName)
			if !ok || check.Status != tt.wantRequiredStatus {
				t.Fatalf("required_capability check = %#v, %v; want %s", check, ok, tt.wantRequiredStatus)
			}
			if !strings.Contains(check.Detail, tt.wantRequiredDetail) {
				t.Fatalf("required_capability detail = %q, want %q", check.Detail, tt.wantRequiredDetail)
			}
		})
	}
}
