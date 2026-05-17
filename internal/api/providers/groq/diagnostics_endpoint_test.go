package groq

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseGroq_EndpointCheckClassifiesConfiguredPaths(t *testing.T) {
	tests := []struct {
		name       string
		apiURL     string
		wantStatus DiagnosticStatus
		wantDetail string
		wantText   string
	}{
		{
			name:       "exact_chat_completions_endpoint",
			apiURL:     "https://example.com" + groqChatCompletionsEndpointPath,
			wantStatus: DiagnosticStatusOK,
			wantDetail: "https://example.com" + groqChatCompletionsEndpointPath,
		},
		{
			name:       "generic_openai_path_warns_currently",
			apiURL:     "https://example.com/v1/chat/completions",
			wantStatus: DiagnosticStatusWarn,
			wantDetail: "https://example.com/v1/chat/completions",
			wantText:   groqChatCompletionsEndpointPath,
		},
		{
			name:       "generic_proxy_path_warns",
			apiURL:     "https://example.com/proxy",
			wantStatus: DiagnosticStatusWarn,
			wantDetail: "https://example.com/proxy",
			wantText:   "intentional proxy endpoint",
		},
		{
			name:       "invalid_url_fails",
			apiURL:     "not a url",
			wantStatus: DiagnosticStatusFail,
			wantDetail: "not a url",
			wantText:   defaultGroqURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(groqAPIKeyEnv, "gsk-test")
			t.Setenv(groqAPIURLEnv, tt.apiURL)

			report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
			endpoint := requireGroqDiagnosticCheckStatus(t, report, "endpoint", tt.wantStatus)
			if endpoint.Detail != tt.wantDetail {
				t.Fatalf("endpoint detail = %q, want %q", endpoint.Detail, tt.wantDetail)
			}
			if tt.wantText != "" && !strings.Contains(endpoint.Message+endpoint.Suggestion, tt.wantText) {
				t.Fatalf("endpoint check = %#v, want text containing %q", endpoint, tt.wantText)
			}
		})
	}
}

func TestDiagnoseGroq_EndpointFailureSkipsSmoke(t *testing.T) {
	t.Setenv(groqAPIKeyEnv, "gsk-test")
	t.Setenv(groqAPIURLEnv, "not a url")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:    config.DefaultConfig(),
		RunSmoke:  true,
		TextSmoke: true,
	})

	requireGroqDiagnosticCheckStatus(t, report, "endpoint", DiagnosticStatusFail)
	smoke := requireGroqDiagnosticCheckStatus(t, report, "smoke", DiagnosticStatusWarn)
	if smoke.Message != "live Groq smoke was skipped because prerequisite checks failed" {
		t.Fatalf("smoke check = %#v, want prerequisite skip", smoke)
	}
	if report.Smoke != nil {
		t.Fatalf("Smoke = %#v, want nil when endpoint check fails", report.Smoke)
	}
}

func TestDiagnoseGroq_ProxyEndpointWarnsAndPrintRequestDoesNotSendNetwork(t *testing.T) {
	server, requestCount := newGroqUnexpectedRequestServer(t)

	t.Setenv(groqAPIKeyEnv, "")
	t.Setenv(groqAPIURLEnv, server.URL+"/proxy")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "meta-llama/llama-4-scout-17b-16e-instruct",
		CatalogModel: "meta-llama/llama-4-scout-17b-16e-instruct",
		PrintRequest: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want proxy endpoint warning only: %#v", report.Checks)
	}
	if requests := requestCount(); requests != 0 {
		t.Fatalf("network requests = %d, want 0", requests)
	}
	requireGroqDiagnosticCheckStatus(t, report, "endpoint", DiagnosticStatusWarn)
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 1 {
		t.Fatalf("RequestPreview = %#v, want one text request", report.RequestPreview)
	}
	if got := report.RequestPreview.Requests[0].URL; got != server.URL+"/proxy" {
		t.Fatalf("preview URL = %q, want proxy URL", got)
	}
}

func requireGroqDiagnosticCheckStatus(t *testing.T, report DiagnosticReport, name string, status DiagnosticStatus) DiagnosticCheck {
	t.Helper()

	check, ok := groqDiagnosticCheckByName(report, name)
	if !ok || check.Status != status {
		t.Fatalf("%s check = %#v, %v; want %s", name, check, ok, status)
	}
	return check
}

func newGroqUnexpectedRequestServer(t *testing.T) (*httptest.Server, func() int) {
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
