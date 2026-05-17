package kimi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseKimi_EndpointCheckClassifiesConfiguredPaths(t *testing.T) {
	tests := []struct {
		name       string
		apiURL     string
		checkName  string
		wantStatus DiagnosticStatus
		wantDetail string
		wantText   string
	}{
		{
			name:       "exact_chat_completions_endpoint",
			apiURL:     "https://example.com" + kimiChatCompletionsEndpointPath,
			checkName:  "api_url",
			wantStatus: DiagnosticStatusOK,
			wantDetail: "https://example.com" + kimiChatCompletionsEndpointPath,
		},
		{
			name:       "generic_proxy_path_warns",
			apiURL:     "https://example.com/proxy",
			checkName:  "api_url_path",
			wantStatus: DiagnosticStatusWarn,
			wantDetail: "https://example.com/proxy",
			wantText:   "intentional proxy endpoint",
		},
		{
			name:       "invalid_url_fails",
			apiURL:     "not a url",
			checkName:  "api_url",
			wantStatus: DiagnosticStatusFail,
			wantDetail: "not a url",
			wantText:   defaultKimiURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setKimiEndpointDiagnosticTestEnv(t, "moonshot-key", tt.apiURL)

			report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
			endpoint := requireKimiDiagnosticCheckStatus(t, report, tt.checkName, tt.wantStatus)
			if endpoint.Detail != tt.wantDetail {
				t.Fatalf("endpoint detail = %q, want %q", endpoint.Detail, tt.wantDetail)
			}
			requireKimiDiagnosticCheckTextContains(t, endpoint, tt.wantText)
		})
	}
}

func TestDiagnoseKimi_EndpointFailureSkipsSmoke(t *testing.T) {
	setKimiEndpointDiagnosticTestEnv(t, "moonshot-key", "not a url")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:    config.DefaultConfig(),
		RunSmoke:  true,
		TextSmoke: true,
	})

	requireKimiDiagnosticCheckStatus(t, report, "api_url", DiagnosticStatusFail)
	smoke := requireKimiDiagnosticCheckStatus(t, report, "smoke", DiagnosticStatusWarn)
	if smoke.Message != "live Kimi smoke was skipped because prerequisite checks failed" {
		t.Fatalf("smoke check = %#v, want prerequisite skip", smoke)
	}
	if report.Smoke != nil {
		t.Fatalf("Smoke = %#v, want nil when endpoint check fails", report.Smoke)
	}
}

func TestDiagnoseKimi_ProxyEndpointWarningStillAllowsLiveSmoke(t *testing.T) {
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
		if r.Header.Get("Authorization") != "Bearer moonshot-key" {
			t.Errorf("Authorization = %q, want Bearer moonshot-key", r.Header.Get("Authorization"))
		}
		writeKimiDiagnosticSSE(t, w,
			`{"choices":[{"delta":{"content":"proxy smoke ok"}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"stop","usage":{"prompt_tokens":7,"completion_tokens":3,"cached_tokens":1}}]}`,
		)
	}))
	defer server.Close()

	setKimiEndpointDiagnosticTestEnv(t, "moonshot-key", server.URL+proxyPath)

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:    config.DefaultConfig(),
		RunSmoke:  true,
		TextSmoke: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want proxy endpoint warning only: %#v", report.Checks)
	}
	requireKimiDiagnosticCheckStatus(t, report, "api_url_path", DiagnosticStatusWarn)
	requireKimiDiagnosticCheckStatus(t, report, "api_url", DiagnosticStatusOK)
	requireKimiDiagnosticCheckStatus(t, report, "smoke", DiagnosticStatusOK)
	wantRequests := int32(len(kimiDiagnosticTextSmokeRequests()))
	if requests.Load() != wantRequests {
		t.Fatalf("requests = %d, want %d live text smoke requests to proxy endpoint", requests.Load(), wantRequests)
	}
	if report.Smoke == nil || report.Smoke.Content != "proxy smoke ok" {
		t.Fatalf("Smoke = %#v, want proxy smoke content", report.Smoke)
	}
}

func TestDiagnoseKimi_ProxyEndpointWarnsAndPrintRequestDoesNotSendNetwork(t *testing.T) {
	server, requestCount := newKimiUnexpectedRequestServer(t)

	setKimiEndpointDiagnosticTestEnv(t, "", server.URL+"/proxy")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		PrintRequest: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want proxy endpoint warning only: %#v", report.Checks)
	}
	if requests := requestCount(); requests != 0 {
		t.Fatalf("network requests = %d, want 0", requests)
	}
	requireKimiDiagnosticCheckStatus(t, report, "api_url_path", DiagnosticStatusWarn)
	requireKimiDiagnosticCheckStatus(t, report, "api_url", DiagnosticStatusOK)
	requireKimiPreviewRequestsUseURL(t, report, server.URL+"/proxy")
}

func newKimiUnexpectedRequestServer(t *testing.T) (*httptest.Server, func() int) {
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

func setKimiEndpointDiagnosticTestEnv(t *testing.T, apiKey, apiURL string) {
	t.Helper()
	t.Setenv(kimiAPIKeyEnv, apiKey)
	t.Setenv(kimiAPIURLEnv, apiURL)
	t.Setenv(kimiFunctionCallingEnv, "")
	t.Setenv("XELYON_MODEL", "")
}

func requireKimiDiagnosticCheckTextContains(t *testing.T, check DiagnosticCheck, want string) {
	t.Helper()
	if want == "" {
		return
	}
	if !strings.Contains(check.Message+check.Suggestion, want) {
		t.Fatalf("diagnostic check = %#v, want text containing %q", check, want)
	}
}

func requireKimiPreviewRequestsUseURL(t *testing.T, report DiagnosticReport, wantURL string) {
	t.Helper()
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) == 0 {
		t.Fatalf("RequestPreview = %#v, want preview requests", report.RequestPreview)
	}
	for _, request := range report.RequestPreview.Requests {
		if request.URL != wantURL {
			t.Fatalf("preview request = %#v, want URL %q", request, wantURL)
		}
	}
}
