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

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	if report.NormalizedBaseURL != proxyBaseURL {
		t.Fatalf("normalized_base_url = %q, want configured proxy base URL", report.NormalizedBaseURL)
	}
	requireDoctorJSONPrintRequestOmittedSmoke(t, report.Smoke)
	requireDoctorJSONProxyWarning(t, report.Checks, "base_url_path", "base_url", proxyBaseURL)
	requireDoctorJSONPrintRequestSkippedAuth(t, report.Checks)
	requireDoctorJSONRequestPreviewRouteAndURL(t, report.RequestPreview, 2, azureprovider.DiagnosticRouteResponsesNonStreaming, proxyBaseURL+"/responses")
}
