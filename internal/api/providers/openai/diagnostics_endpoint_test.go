package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseOpenAI_EndpointURLChecksAcceptOfficialPaths(t *testing.T) {
	apiURL := "https://openai.example" + openAIChatCompletionsEndpointPath
	responsesURL := "https://openai.example" + openAIResponsesEndpointPath

	for _, tc := range []struct {
		name         string
		model        string
		catalogModel string
	}{
		{
			name:         "chat completions route",
			model:        "gpt-4",
			catalogModel: "gpt-4",
		},
		{
			name:         "responses route",
			model:        "gpt-5.5-pro",
			catalogModel: "gpt-5.5-pro",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := diagnoseOpenAIEndpointReport(t, tc.model, tc.catalogModel, apiURL, responsesURL, DiagnosticOptions{})
			if report.HasFailures() {
				t.Fatalf("Diagnose() has failures: %#v", report.Checks)
			}
			requireOpenAIDiagnosticCheck(t, report, "api_url", DiagnosticStatusOK)
			requireOpenAIDiagnosticCheck(t, report, "responses_url", DiagnosticStatusOK)
			requireNoOpenAIDiagnosticCheck(t, report, "api_url_path")
			requireNoOpenAIDiagnosticCheck(t, report, "responses_url_path")
		})
	}
}

func TestDiagnoseOpenAI_EndpointURLChecksWarnForIntentionalProxyPaths(t *testing.T) {
	for _, tc := range []struct {
		name         string
		model        string
		catalogModel string
		apiURL       string
		responsesURL string
		warnCheck    string
		okCheck      string
	}{
		{
			name:         "chat completions proxy path",
			model:        "gpt-4",
			catalogModel: "gpt-4",
			apiURL:       "https://openai.example/proxy/chat",
			warnCheck:    "api_url_path",
			okCheck:      "api_url",
		},
		{
			name:         "responses proxy path",
			model:        "gpt-5.5-pro",
			catalogModel: "gpt-5.5-pro",
			responsesURL: "https://openai.example/proxy/responses",
			warnCheck:    "responses_url_path",
			okCheck:      "responses_url",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := diagnoseOpenAIEndpointReport(t, tc.model, tc.catalogModel, tc.apiURL, tc.responsesURL, DiagnosticOptions{})
			if report.HasFailures() {
				t.Fatalf("Diagnose() has failures: %#v", report.Checks)
			}
			warn := requireOpenAIDiagnosticCheck(t, report, tc.warnCheck, DiagnosticStatusWarn)
			if !strings.Contains(warn.Detail, "https://openai.example/proxy/") ||
				!strings.Contains(warn.Suggestion, "intentional proxy") {
				t.Fatalf("%s check = %#v, want proxy detail and suggestion", tc.warnCheck, warn)
			}
			requireOpenAIDiagnosticCheck(t, report, tc.okCheck, DiagnosticStatusOK)
		})
	}
}

func TestDiagnoseOpenAI_EndpointURLChecksClassifyInvalidURLsByActiveRoute(t *testing.T) {
	for _, tc := range []struct {
		name         string
		model        string
		catalogModel string
		apiURL       string
		responsesURL string
		checkName    string
		wantStatus   DiagnosticStatus
		wantFailures bool
	}{
		{
			name:         "active chat completions endpoint fails",
			model:        "gpt-4",
			catalogModel: "gpt-4",
			apiURL:       "://bad",
			checkName:    "api_url",
			wantStatus:   DiagnosticStatusFail,
			wantFailures: true,
		},
		{
			name:         "inactive responses endpoint warns on chat completions route",
			model:        "gpt-4",
			catalogModel: "gpt-4",
			responsesURL: "://bad",
			checkName:    "responses_url",
			wantStatus:   DiagnosticStatusWarn,
		},
		{
			name:         "inactive chat completions endpoint warns on responses route",
			model:        "gpt-5.5-pro",
			catalogModel: "gpt-5.5-pro",
			apiURL:       "://bad",
			checkName:    "api_url",
			wantStatus:   DiagnosticStatusWarn,
		},
		{
			name:         "active responses endpoint fails",
			model:        "gpt-5.5-pro",
			catalogModel: "gpt-5.5-pro",
			responsesURL: "://bad",
			checkName:    "responses_url",
			wantStatus:   DiagnosticStatusFail,
			wantFailures: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := diagnoseOpenAIEndpointReport(t, tc.model, tc.catalogModel, tc.apiURL, tc.responsesURL, DiagnosticOptions{})
			if report.HasFailures() != tc.wantFailures {
				t.Fatalf("HasFailures() = %t, want %t: %#v", report.HasFailures(), tc.wantFailures, report.Checks)
			}
			requireOpenAIDiagnosticCheck(t, report, tc.checkName, tc.wantStatus)
		})
	}
}

func TestDiagnoseOpenAI_RemoteModesKeepEndpointChecksWithCapabilityFlags(t *testing.T) {
	tests := []struct {
		name    string
		options DiagnosticOptions
	}{
		{
			name: "print request",
			options: DiagnosticOptions{
				Capabilities: true,
				PrintRequest: true,
			},
		},
		{
			name: "smoke",
			options: DiagnosticOptions{
				RequiredCapabilities: []string{"responses_api"},
				RunSmoke:             true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := diagnoseOpenAIEndpointReport(t, "gpt-5.4", "gpt-5.4", "", "://bad", tt.options)
			if !report.HasFailures() {
				t.Fatalf("HasFailures() = false, want active responses endpoint failure: %#v", report.Checks)
			}
			requireOpenAIDiagnosticCheck(t, report, "responses_url", DiagnosticStatusFail)
		})
	}
}

func TestDiagnoseOpenAI_ChatCompletionsSmokeUsesConfiguredProxyEndpoint(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Fatalf("Authorization = %q, want Bearer sk-test", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"xelyon openai doctor ok"}}]}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":7,"completion_tokens":4}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: [DONE]`)
		fmt.Fprintln(w)
	}))
	defer server.Close()

	proxyPath := "/proxy/chat"
	report := diagnoseOpenAIEndpointReport(t, "gpt-4", "gpt-4", server.URL+proxyPath, "", DiagnosticOptions{
		RunSmoke: true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	requireOpenAIDiagnosticCheck(t, report, "api_url_path", DiagnosticStatusWarn)
	requireOpenAIDiagnosticCheck(t, report, "api_url", DiagnosticStatusOK)
	if receivedPath != proxyPath {
		t.Fatalf("request path = %q, want configured proxy path %q", receivedPath, proxyPath)
	}
	if report.Smoke == nil || !report.Smoke.UsageObserved {
		t.Fatalf("Smoke = %#v, want successful chat completions smoke with usage", report.Smoke)
	}
}

func TestDiagnoseOpenAI_ResponsesSmokeUsesConfiguredProxyEndpoint(t *testing.T) {
	var received struct {
		Path string
		Body map[string]any
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&received.Body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_doctor","output_text":"xelyon openai doctor ok","usage":{"input_tokens":10,"output_tokens":6}}`))
	}))
	defer server.Close()

	proxyPath := "/proxy/responses"
	report := diagnoseOpenAIEndpointReport(t, "gpt-5.5-pro", "gpt-5.5-pro", "", server.URL+proxyPath, DiagnosticOptions{
		RunSmoke: true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	requireOpenAIDiagnosticCheck(t, report, "responses_url_path", DiagnosticStatusWarn)
	requireOpenAIDiagnosticCheck(t, report, "responses_url", DiagnosticStatusOK)
	if received.Path != proxyPath {
		t.Fatalf("request path = %q, want configured proxy path %q", received.Path, proxyPath)
	}
	if received.Body["store"] != false {
		t.Fatalf("store = %#v, want false for doctor smoke", received.Body["store"])
	}
	if report.Smoke == nil || report.Smoke.ResponseID != "resp_doctor" {
		t.Fatalf("Smoke = %#v, want successful responses smoke with response ID", report.Smoke)
	}
}

func diagnoseOpenAIEndpointReport(t *testing.T, model, catalogModel, apiURL, responsesURL string, options DiagnosticOptions) DiagnosticReport {
	t.Helper()
	t.Setenv(openAIAPIKeyEnv, "sk-test")
	t.Setenv(openAIAPIURLEnv, apiURL)
	t.Setenv(openAIResponsesURLEnv, responsesURL)
	t.Setenv("OPENAI_FUNCTION_CALLING", "1")
	if options.Config == nil {
		options.Config = config.DefaultConfig()
	}
	options.Model = model
	options.CatalogModel = catalogModel
	return Diagnose(context.Background(), options)
}

func requireOpenAIDiagnosticCheck(t *testing.T, report DiagnosticReport, name string, status DiagnosticStatus) DiagnosticCheck {
	t.Helper()
	check, ok := openAIDiagnosticCheckByName(report, name)
	if !ok || check.Status != status {
		t.Fatalf("%s check = %#v, %v; want %s", name, check, ok, status)
	}
	return check
}

func requireNoOpenAIDiagnosticCheck(t *testing.T, report DiagnosticReport, name string) {
	t.Helper()
	if check, ok := openAIDiagnosticCheckByName(report, name); ok {
		t.Fatalf("%s check = %#v, want omitted", name, check)
	}
}
