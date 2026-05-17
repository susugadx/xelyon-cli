package claude

import "testing"

func setClaudeDiagnosticTestEnv(t *testing.T, apiURL, apiKey string) {
	t.Helper()
	t.Setenv(anthropicAPIKeyEnv, apiKey)
	t.Setenv(anthropicAPIURLEnv, apiURL)
	t.Setenv(claudeFunctionCallEnv, "")
	t.Setenv("XELYON_MODEL", "")
}

func requireClaudeToolChoice(t *testing.T, choice *ClaudeToolChoice, name string) {
	t.Helper()
	if choice == nil {
		t.Fatalf("ToolChoice = nil, want forced %s tool choice", name)
	}
	if choice.Type != "tool" || choice.Name != name {
		t.Fatalf("ToolChoice = %+v, want forced %s tool choice", choice, name)
	}
}

func requireClaudePreviewRequestsUseURL(t *testing.T, report DiagnosticReport, wantURL string) {
	t.Helper()
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) == 0 {
		t.Fatalf("RequestPreview = %#v, want preview requests", report.RequestPreview)
	}
	for _, request := range report.RequestPreview.Requests {
		if request.Skipped {
			continue
		}
		if request.URL != wantURL {
			t.Fatalf("%s preview URL = %q, want %q", request.Name, request.URL, wantURL)
		}
	}
}
