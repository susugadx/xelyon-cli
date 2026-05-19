package openrouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseOpenRouter_AnthropicMessagesEndpointOverrideFailsAndSkipsSmoke(t *testing.T) {
	server, requestCount := newOpenRouterUnexpectedRequestServer(t)

	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, server.URL+"/v1/messages")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "anthropic/claude-sonnet-4.6",
		CatalogModel: "anthropic/claude-sonnet-4.6",
		RunSmoke:     true,
		TextSmoke:    true,
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want endpoint path failure: %#v", report.Checks)
	}
	if requests := requestCount(); requests != 0 {
		t.Fatalf("network requests = %d, want smoke skipped before request", requests)
	}
	endpoint := requireOpenRouterDiagnosticCheckStatus(t, report, "endpoint", DiagnosticStatusFail)
	if endpoint.Detail != server.URL+"/v1/messages" {
		t.Fatalf("endpoint detail = %q, want configured messages endpoint", endpoint.Detail)
	}
	smoke := requireOpenRouterDiagnosticCheckStatus(t, report, "smoke", DiagnosticStatusWarn)
	if smoke.Message != "live OpenRouter smoke was skipped because prerequisite checks failed" {
		t.Fatalf("smoke check = %#v, want prerequisite skip", smoke)
	}
	if report.Smoke != nil {
		t.Fatalf("Smoke = %#v, want nil when endpoint check fails", report.Smoke)
	}
}

func TestDiagnoseOpenRouter_ChatRouteRejectsMessagesEndpointOverride(t *testing.T) {
	server, _ := newOpenRouterUnexpectedRequestServer(t)

	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, server.URL+"/v1/messages")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "openai/gpt-5.4",
		CatalogModel: "openai/gpt-5.4",
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want endpoint path failure: %#v", report.Checks)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "endpoint", DiagnosticStatusFail)
}

func TestDiagnoseOpenRouter_NonstandardProxyPathStillWarnsAndPrintsDerivedAnthropicURL(t *testing.T) {
	server, requestCount := newOpenRouterUnexpectedRequestServer(t)

	t.Setenv(openRouterAPIKeyEnv, "")
	t.Setenv(openRouterAPIURLEnv, server.URL+"/proxy")
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "anthropic/claude-sonnet-4.6",
		CatalogModel: "anthropic/claude-sonnet-4.6",
		ToolSmoke:    true,
		PrintRequest: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want proxy path warning only: %#v", report.Checks)
	}
	if requests := requestCount(); requests != 0 {
		t.Fatalf("network requests = %d, want 0", requests)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "endpoint", DiagnosticStatusWarn)
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 1 {
		t.Fatalf("RequestPreview = %#v, want one tool request", report.RequestPreview)
	}
	preview := report.RequestPreview.Requests[0]
	if preview.URL != server.URL+"/proxy/messages" {
		t.Fatalf("preview URL = %q, want proxy-derived Anthropic URL", preview.URL)
	}
}

func newOpenRouterUnexpectedRequestServer(t *testing.T) (*httptest.Server, func() int) {
	t.Helper()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "unexpected network request", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	return server, func() int {
		return int(requests.Load())
	}
}
