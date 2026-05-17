package claude

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseClaude_EndpointCheckClassifiesConfiguredPaths(t *testing.T) {
	tests := []struct {
		name       string
		apiURL     string
		wantStatus DiagnosticStatus
		wantDetail string
		wantText   string
	}{
		{
			name:       "built_in_endpoint",
			apiURL:     "",
			wantStatus: DiagnosticStatusOK,
			wantDetail: defaultClaudeURL,
		},
		{
			name:       "exact_messages_endpoint",
			apiURL:     "https://example.com" + claudeMessagesEndpointPath,
			wantStatus: DiagnosticStatusOK,
			wantDetail: "https://example.com" + claudeMessagesEndpointPath,
		},
		{
			name:       "generic_proxy_path_warns",
			apiURL:     "https://example.com/proxy",
			wantStatus: DiagnosticStatusWarn,
			wantDetail: "https://example.com/proxy",
			wantText:   "intentional proxy endpoint",
		},
		{
			name:       "legacy_anthropic_complete_path_warns",
			apiURL:     "https://api.anthropic.com/v1/complete",
			wantStatus: DiagnosticStatusWarn,
			wantDetail: "https://api.anthropic.com/v1/complete",
			wantText:   claudeMessagesEndpointPath,
		},
		{
			name:       "invalid_url_fails",
			apiURL:     "not a url",
			wantStatus: DiagnosticStatusFail,
			wantDetail: "not a url",
			wantText:   defaultClaudeURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setClaudeDiagnosticTestEnv(t, tt.apiURL, "claude-key")

			report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
			endpoint := requireClaudeDiagnosticCheckStatus(t, report, "endpoint", tt.wantStatus)
			if endpoint.Detail != tt.wantDetail {
				t.Fatalf("endpoint detail = %q, want %q", endpoint.Detail, tt.wantDetail)
			}
			requireClaudeDiagnosticCheckTextContains(t, endpoint, tt.wantText)
		})
	}
}

func TestDiagnoseClaude_EndpointFailureSkipsSmoke(t *testing.T) {
	setClaudeDiagnosticTestEnv(t, "not a url", "claude-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:    config.DefaultConfig(),
		RunSmoke:  true,
		TextSmoke: true,
	})

	requireClaudeDiagnosticCheckStatus(t, report, "endpoint", DiagnosticStatusFail)
	smoke := requireClaudeDiagnosticCheckStatus(t, report, "smoke", DiagnosticStatusWarn)
	if smoke.Message != "live Claude smoke was skipped because prerequisite checks failed" {
		t.Fatalf("smoke check = %#v, want prerequisite skip", smoke)
	}
	if report.Smoke != nil {
		t.Fatalf("Smoke = %#v, want nil when endpoint check fails", report.Smoke)
	}
}

func TestDiagnoseClaude_ProxyEndpointWarningStillAllowsLiveSmoke(t *testing.T) {
	const proxyPath = "/proxy"
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != proxyPath {
			t.Errorf("path = %s, want %s", r.URL.Path, proxyPath)
		}
		if r.Header.Get("x-api-key") != "claude-key" {
			t.Errorf("x-api-key = %q, want claude-key", r.Header.Get("x-api-key"))
		}
		writeClaudeDiagnosticJSON(t, w, []Content{{Type: "text", Text: "proxy smoke ok"}}, StreamUsage{InputTokens: 7, OutputTokens: 3})
	}))
	defer server.Close()

	setClaudeDiagnosticTestEnv(t, server.URL+proxyPath, "claude-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultClaudeModel,
		CatalogModel: defaultClaudeModel,
		RunSmoke:     true,
		TextSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want proxy endpoint warning only: %#v", report.Checks)
	}
	requireClaudeDiagnosticCheckStatus(t, report, "endpoint", DiagnosticStatusWarn)
	requireClaudeDiagnosticCheckStatus(t, report, "smoke", DiagnosticStatusOK)
	if requests.Load() != 1 {
		t.Fatalf("requests = %d, want one live smoke request to proxy endpoint", requests.Load())
	}
	if report.Smoke == nil || report.Smoke.Content != "proxy smoke ok" {
		t.Fatalf("Smoke = %#v, want proxy smoke content", report.Smoke)
	}
}

func TestDiagnoseClaude_ProxyEndpointWarnsAndPrintRequestDoesNotSendNetwork(t *testing.T) {
	server, requestCount := newClaudeUnexpectedRequestServer(t)

	setClaudeDiagnosticTestEnv(t, server.URL+"/proxy", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultClaudeModel,
		CatalogModel: defaultClaudeModel,
		PrintRequest: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want proxy endpoint warning only: %#v", report.Checks)
	}
	if requests := requestCount(); requests != 0 {
		t.Fatalf("network requests = %d, want 0", requests)
	}
	requireClaudeDiagnosticCheckStatus(t, report, "endpoint", DiagnosticStatusWarn)
	requireClaudePreviewRequestsUseURL(t, report, server.URL+"/proxy")
}

func newClaudeUnexpectedRequestServer(t *testing.T) (*httptest.Server, func() int) {
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

func requireClaudeDiagnosticCheckTextContains(t *testing.T, check DiagnosticCheck, want string) {
	t.Helper()
	if want == "" {
		return
	}
	if !strings.Contains(check.Message+check.Suggestion, want) {
		t.Fatalf("diagnostic check = %#v, want text containing %q", check, want)
	}
}
