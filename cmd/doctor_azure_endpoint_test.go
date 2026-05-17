package cmd

import (
	"strings"
	"testing"

	azureprovider "github.com/susugadx/xelyon-cli/internal/api/providers/azure"
)

func TestRootCommand_AzureDoctorHelpShowsEndpointContract(t *testing.T) {
	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "azure", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	for _, want := range []string{
		"AZURE_OPENAI_BASE_URL",
		"/openai/v1",
		"<normalized_base_url>/responses",
		"intentional",
		"proxy",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output = %q, want endpoint contract substring %q", out.String(), want)
		}
	}
}

func TestRunAzureDoctorInvocation_PrintRequestJSONReportsProxyBaseURLWarning(t *testing.T) {
	proxyBaseURL := "https://azure.example/proxy/azure"
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AZURE_OPENAI_BASE_URL", proxyBaseURL)
	t.Setenv("AZURE_OPENAI_API_KEY", "")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN", "")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND", "")

	cmd, out := newDoctorSubcommandTest(t, newAzureDoctorCommand)

	doctorDeploymentFlag = "corp-gpt55-pro-deployment"
	doctorCatalogModelFlag = "gpt-5.5-pro"
	doctorAzureRetentionSmokeFlag = true
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runAzureDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runAzureDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[struct {
		NormalizedBaseURL string `json:"normalized_base_url"`
		Smoke             any    `json:"smoke"`
		RequestPreview    struct {
			Requests []struct {
				Route string `json:"route"`
				URL   string `json:"url"`
			} `json:"requests"`
		} `json:"request_preview"`
		Checks []doctorJSONCheck `json:"checks"`
	}](t, out)
	if report.NormalizedBaseURL != proxyBaseURL {
		t.Fatalf("normalized_base_url = %q, want configured proxy base URL", report.NormalizedBaseURL)
	}
	if report.Smoke != nil {
		t.Fatalf("smoke = %#v, want omitted for --print-request", report.Smoke)
	}
	endpoint := requireDoctorJSONCheck(t, report.Checks, "base_url_path")
	requireDoctorJSONCheckStatus(t, endpoint, "warn")
	requireDoctorJSONCheckDetailContains(t, endpoint, proxyBaseURL)
	requireDoctorJSONCheckSuggestionContains(t, endpoint, "intentional proxy")
	requireDoctorJSONCheckStatus(t, requireDoctorJSONCheck(t, report.Checks, "base_url"), "ok")
	requireNoDoctorJSONChecks(t, report.Checks, "auth")
	if len(report.RequestPreview.Requests) != 2 {
		t.Fatalf("request_preview = %#v, want two retention requests", report.RequestPreview)
	}
	for _, request := range report.RequestPreview.Requests {
		if request.Route != azureprovider.DiagnosticRouteResponsesNonStreaming ||
			request.URL != proxyBaseURL+"/responses" {
			t.Fatalf("request_preview = %#v, want proxy responses request URLs", report.RequestPreview)
		}
	}
}
