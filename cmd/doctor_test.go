package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	azureprovider "github.com/susugadx/xelyon-cli/internal/api/providers/azure"
)

func TestRunAzureDoctorInvocation_JSONReportsConfiguredDeployment(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com/openai/v1")
	t.Setenv("AZURE_OPENAI_API_KEY", "azure-key")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN", "")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND", "")

	cmd, out := newDoctorSubcommandTest(t, newAzureDoctorCommand)

	doctorDeploymentFlag = "corp-codex-deployment"
	doctorCatalogModelFlag = "gpt-5.3-codex"
	doctorJSONFlag = true

	if err := runAzureDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runAzureDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[struct {
		Provider          string            `json:"provider"`
		Deployment        string            `json:"deployment"`
		CatalogModel      string            `json:"catalog_model"`
		Route             string            `json:"route"`
		RouteReason       string            `json:"route_reason"`
		NormalizedBaseURL string            `json:"normalized_base_url"`
		Checks            []doctorJSONCheck `json:"checks"`
	}](t, out)
	if report.Provider != "azure" {
		t.Fatalf("provider = %q, want azure", report.Provider)
	}
	if report.Deployment != "corp-codex-deployment" {
		t.Fatalf("deployment = %q, want CLI deployment", report.Deployment)
	}
	if report.CatalogModel != "gpt-5.3-codex" {
		t.Fatalf("catalog_model = %q, want CLI catalog model", report.CatalogModel)
	}
	if report.Route != "responses_streaming" {
		t.Fatalf("route = %q, want responses_streaming", report.Route)
	}
	if !strings.Contains(report.RouteReason, "catalog_model=gpt-5.3-codex supports Responses streaming") {
		t.Fatalf("route_reason = %q, want catalog streaming reason", report.RouteReason)
	}
	if report.NormalizedBaseURL != "https://example.openai.azure.com/openai/v1" {
		t.Fatalf("normalized_base_url = %q, want v1 URL", report.NormalizedBaseURL)
	}
	catalogPolicy := requireDoctorJSONCheck(t, report.Checks, "catalog_policy")
	requireDoctorJSONCheckStatus(t, catalogPolicy, "ok")
	requireDoctorJSONCheckDetailContains(t, catalogPolicy, "max_output_tokens=128000")
}

func TestRunAzureDoctorInvocation_PrintRequestJSONDoesNotRequireAuth(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com/openai/v1")
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
		Smoke          any `json:"smoke"`
		RequestPreview struct {
			Requests []struct {
				Name               string `json:"name"`
				RetentionPayload   bool   `json:"retention_payload"`
				PreviousResponseID string `json:"previous_response_id"`
				URL                string `json:"url"`
				Body               struct {
					Model              string `json:"model"`
					Store              bool   `json:"store"`
					PreviousResponseID string `json:"previous_response_id"`
				} `json:"body"`
			} `json:"requests"`
		} `json:"request_preview"`
	}](t, out)
	if report.Smoke != nil {
		t.Fatalf("smoke = %#v, want omitted for --print-request", report.Smoke)
	}
	if len(report.RequestPreview.Requests) != 2 {
		t.Fatalf("request_preview = %#v, want two retention requests", report.RequestPreview)
	}
	followup := report.RequestPreview.Requests[1]
	if followup.Name != "retention_followup" || !followup.RetentionPayload {
		t.Fatalf("followup preview = %#v, want retention followup", followup)
	}
	if followup.URL != "https://example.openai.azure.com/openai/v1/responses" {
		t.Fatalf("followup URL = %q, want Azure Responses endpoint", followup.URL)
	}
	if followup.Body.Model != "corp-gpt55-pro-deployment" || !followup.Body.Store || followup.Body.PreviousResponseID != followup.PreviousResponseID {
		t.Fatalf("followup body = %#v, want deployment, store true, and placeholder previous_response_id", followup.Body)
	}
}

func TestRunAzureDoctorInvocation_CapabilitiesJSONDoesNotRequireEndpointOrAuth(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_ = os.Unsetenv("AZURE_OPENAI_BASE_URL")
	_ = os.Unsetenv("AZURE_OPENAI_API_KEY")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND")

	cmd, out := newDoctorSubcommandTest(t, newAzureDoctorCommand)

	doctorDeploymentFlag = "corp-codex-deployment"
	doctorCatalogModelFlag = "gpt-5.3-codex"
	doctorCapabilitiesFlag = true
	doctorJSONFlag = true

	if err := runAzureDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runAzureDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[struct {
		Capabilities struct {
			Deployment         string `json:"deployment"`
			CatalogModel       string `json:"catalog_model"`
			ResponsesAPI       bool   `json:"responses_api"`
			ResponsesStreaming bool   `json:"responses_streaming"`
			ServerCompaction   struct {
				RequestPayload   bool `json:"request_payload"`
				CompactThreshold int  `json:"compact_threshold"`
			} `json:"server_compaction"`
		} `json:"capabilities"`
		Checks []doctorJSONCheck `json:"checks"`
	}](t, out)
	if report.Capabilities.Deployment != "corp-codex-deployment" ||
		report.Capabilities.CatalogModel != "gpt-5.3-codex" ||
		!report.Capabilities.ResponsesAPI ||
		!report.Capabilities.ResponsesStreaming ||
		!report.Capabilities.ServerCompaction.RequestPayload ||
		report.Capabilities.ServerCompaction.CompactThreshold <= 0 {
		t.Fatalf("capabilities = %+v, want resolved Azure capabilities", report.Capabilities)
	}
	requireNoDoctorJSONChecks(t, report.Checks, "base_url", "auth")
}

func TestRunAzureDoctorInvocation_RequireCapabilityFailsWithoutEndpointOrAuthCheck(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_ = os.Unsetenv("AZURE_OPENAI_BASE_URL")
	_ = os.Unsetenv("AZURE_OPENAI_API_KEY")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND")

	cmd, out := newDoctorSubcommandTest(t, newAzureDoctorCommand)

	doctorDeploymentFlag = "corp-gpt55-pro-deployment"
	doctorCatalogModelFlag = "gpt-5.5-pro"
	doctorRequiredCapabilityFlags = []string{"responses_streaming"}
	doctorJSONFlag = true

	err := runAzureDoctorInvocation(cmd, nil)
	if err == nil {
		t.Fatalf("runAzureDoctorInvocation() error = nil, want required capability failure\noutput:\n%s", out.String())
	}

	report := unmarshalDoctorJSON[struct {
		Checks []doctorJSONCheck `json:"checks"`
	}](t, out)
	requireNoDoctorJSONChecks(t, report.Checks, "base_url", "auth")
	check := requireDoctorJSONCheck(t, report.Checks, "required_capability")
	requireDoctorJSONCheckStatus(t, check, "fail")
	requireDoctorJSONCheckDetailContains(t, check, "responses_streaming=missing")
}

func TestRunAzureDoctorInvocation_RequireStreamingCapabilityFailsWithoutCatalogModel(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_ = os.Unsetenv("AZURE_OPENAI_BASE_URL")
	_ = os.Unsetenv("AZURE_OPENAI_API_KEY")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND")

	cmd, out := newDoctorSubcommandTest(t, newAzureDoctorCommand)

	doctorDeploymentFlag = "corp-gpt55-pro-deployment"
	doctorRequiredCapabilityFlags = []string{"responses_streaming"}
	doctorJSONFlag = true

	err := runAzureDoctorInvocation(cmd, nil)
	if err == nil {
		t.Fatalf("runAzureDoctorInvocation() error = nil, want unknown required capability failure\noutput:\n%s", out.String())
	}

	report := unmarshalDoctorJSON[struct {
		Checks []doctorJSONCheck `json:"checks"`
	}](t, out)
	requireNoDoctorJSONChecks(t, report.Checks, "base_url", "auth")
	check := requireDoctorJSONCheck(t, report.Checks, "required_capability")
	requireDoctorJSONCheckStatus(t, check, "fail")
	requireDoctorJSONCheckDetailContains(t, check, "responses_streaming=unknown")
	requireDoctorJSONCheckSuggestionContains(t, check, "--catalog-model")
}

func TestRunAzureDoctorInvocation_RequireStreamingCapabilityPassesWithKnownFallbackCatalogModel(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_ = os.Unsetenv("AZURE_OPENAI_BASE_URL")
	_ = os.Unsetenv("AZURE_OPENAI_API_KEY")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND")

	cmd, out := newDoctorSubcommandTest(t, newAzureDoctorCommand)

	doctorDeploymentFlag = "gpt-5.4"
	doctorRequiredCapabilityFlags = []string{"responses_streaming"}
	doctorJSONFlag = true

	err := runAzureDoctorInvocation(cmd, nil)
	if err != nil {
		t.Fatalf("runAzureDoctorInvocation() error = %v, want nil for known fallback catalog model\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[struct {
		Checks []doctorJSONCheck `json:"checks"`
	}](t, out)
	requireNoDoctorJSONChecks(t, report.Checks, "base_url", "auth")
	check := requireDoctorJSONCheck(t, report.Checks, "required_capability")
	requireDoctorJSONCheckStatus(t, check, "ok")
	requireDoctorJSONCheckDetailContains(t, check, "responses_streaming=ok")
}

func TestRunAzureDoctorInvocation_FailsForMissingAzureSetup(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_ = os.Unsetenv("AZURE_OPENAI_BASE_URL")
	_ = os.Unsetenv("AZURE_OPENAI_API_KEY")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND")

	cmd, out := newDoctorSubcommandTest(t, newAzureDoctorCommand)

	err := runAzureDoctorInvocation(cmd, nil)
	if err == nil {
		t.Fatalf("runAzureDoctorInvocation() error = nil, want diagnostics failure\noutput:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "base_url") {
		t.Fatalf("output = %q, want base_url failure", out.String())
	}
	if !strings.Contains(out.String(), "deployment") {
		t.Fatalf("output = %q, want deployment failure", out.String())
	}
}

func TestRunAzureDoctorInvocation_PrintConfigDoesNotRequireAzureEnv(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_ = os.Unsetenv("AZURE_OPENAI_BASE_URL")
	_ = os.Unsetenv("AZURE_OPENAI_API_KEY")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND")

	cmd, out := newDoctorSubcommandTest(t, newAzureDoctorCommand)

	doctorDeploymentFlag = "corp-codex-deployment"
	doctorCatalogModelFlag = "gpt-5.3-codex"
	doctorPrintConfigFlag = true

	if err := runAzureDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runAzureDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	output := out.String()
	for _, want := range []string{
		"default_provider: azure",
		"provider_models:",
		"default_model: corp-codex-deployment",
		"catalog_model: gpt-5.3-codex",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}
	if strings.Contains(output, "Azure OpenAI doctor") {
		t.Fatalf("output = %q, should only print YAML snippet", output)
	}
}

func TestRunAzureDoctorInvocation_PrintConfigRequiresDeploymentAndCatalogModel(t *testing.T) {
	cmd, _ := newDoctorSubcommandTest(t, newAzureDoctorCommand)

	doctorPrintConfigFlag = true
	doctorDeploymentFlag = "corp-gpt55-deployment"

	err := runAzureDoctorInvocation(cmd, nil)
	if err == nil {
		t.Fatal("runAzureDoctorInvocation() error = nil, want missing catalog-model error")
	}
	if !strings.Contains(err.Error(), "--catalog-model") {
		t.Fatalf("error = %v, want --catalog-model guidance", err)
	}
	if !cmd.SilenceUsage {
		t.Fatal("cmd.SilenceUsage = false, want true for print-config validation error")
	}
}

func TestRunAzureDoctorInvocation_PrintConfigRejectsCapabilities(t *testing.T) {
	cmd, _ := newDoctorSubcommandTest(t, newAzureDoctorCommand)

	doctorPrintConfigFlag = true
	doctorCapabilitiesFlag = true
	doctorDeploymentFlag = "corp-gpt55-deployment"
	doctorCatalogModelFlag = "gpt-5.5"

	err := runAzureDoctorInvocation(cmd, nil)
	if err == nil {
		t.Fatal("runAzureDoctorInvocation() error = nil, want print-config capabilities conflict")
	}
	if !strings.Contains(err.Error(), "--capabilities") {
		t.Fatalf("error = %v, want --capabilities conflict guidance", err)
	}
	if !cmd.SilenceUsage {
		t.Fatal("cmd.SilenceUsage = false, want true for print-config validation error")
	}
}

func TestRunAzureDoctorInvocation_PrintConfigRejectsRequiredCapability(t *testing.T) {
	cmd, _ := newDoctorSubcommandTest(t, newAzureDoctorCommand)

	doctorPrintConfigFlag = true
	doctorRequiredCapabilityFlags = []string{"responses_api"}
	doctorDeploymentFlag = "corp-gpt55-deployment"
	doctorCatalogModelFlag = "gpt-5.5"

	err := runAzureDoctorInvocation(cmd, nil)
	if err == nil {
		t.Fatal("runAzureDoctorInvocation() error = nil, want print-config required capability conflict")
	}
	if !strings.Contains(err.Error(), "--require-capability") {
		t.Fatalf("error = %v, want --require-capability conflict guidance", err)
	}
	if !cmd.SilenceUsage {
		t.Fatal("cmd.SilenceUsage = false, want true for print-config validation error")
	}
}

func TestRootCommand_AzureDoctorCommandParsesFlags(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com/openai/v1")
	t.Setenv("AZURE_OPENAI_API_KEY", "azure-key")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN", "")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND", "")

	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "azure", "--deployment", "corp-gpt55-deployment", "--catalog-model", "gpt-5.5", "--json"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"deployment": "corp-gpt55-deployment"`) {
		t.Fatalf("output = %q, want parsed deployment", out.String())
	}
}

func TestRootCommand_AzureDoctorHelpShowsDoctorFlags(t *testing.T) {
	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "azure", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "--deployment") {
		t.Fatalf("output = %q, want Azure doctor flags", out.String())
	}
	if !strings.Contains(out.String(), "--print-config") {
		t.Fatalf("output = %q, want print-config flag", out.String())
	}
	if !strings.Contains(out.String(), "--retention-smoke") {
		t.Fatalf("output = %q, want retention smoke flag", out.String())
	}
	if !strings.Contains(out.String(), "--capabilities") {
		t.Fatalf("output = %q, want capabilities flag", out.String())
	}
	if !strings.Contains(out.String(), "--require-capability") {
		t.Fatalf("output = %q, want require capability flag", out.String())
	}
	if !strings.Contains(out.String(), "--print-request") {
		t.Fatalf("output = %q, want print-request flag", out.String())
	}
	if !strings.Contains(out.String(), "Diagnose Azure OpenAI configuration") {
		t.Fatalf("output = %q, want Azure doctor help", out.String())
	}
	if strings.Contains(out.String(), "XELYON CLI is an AI coding agent") {
		t.Fatalf("output = %q, should not show root long help", out.String())
	}
}

func TestRootCommand_AzureDoctorFailureDoesNotPrintRootUsage(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_ = os.Unsetenv("AZURE_OPENAI_BASE_URL")
	_ = os.Unsetenv("AZURE_OPENAI_API_KEY")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND")

	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "azure"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("root Execute() error = nil, want diagnostics failure\noutput:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "Azure OpenAI doctor") {
		t.Fatalf("output = %q, want doctor report", out.String())
	}
	if strings.Contains(out.String(), "Usage:\n  xelyon [query]") {
		t.Fatalf("output = %q, should not append root usage", out.String())
	}
}

func TestRenderAzureDoctorTextIncludesSmokeObservability(t *testing.T) {
	report := azureprovider.DiagnosticReport{
		Provider:    "azure",
		Route:       azureprovider.DiagnosticRouteResponsesStreaming,
		RouteReason: "deployment=corp-codex uses Responses API; catalog_model=gpt-5.3-codex supports Responses streaming",
		Checks: []azureprovider.DiagnosticCheck{
			{Name: "smoke", Status: azureprovider.DiagnosticStatusOK, Message: "live Azure OpenAI smoke request succeeded"},
		},
		Smoke: &azureprovider.DiagnosticSmokeResult{
			Ran:           true,
			Content:       "xelyon azure doctor ok",
			ResponseID:    "resp_text",
			Duration:      "1ms",
			UsageObserved: true,
			Usage: azureprovider.DiagnosticSmokeUsage{
				InputTokens:         10,
				OutputTokens:        4,
				ThinkingTokens:      2,
				CachedInputTokens:   3,
				CacheCreationTokens: 1,
			},
			Cost: azureprovider.DiagnosticSmokeCost{
				USD: 0.00012345,
			},
		},
	}

	var out bytes.Buffer
	renderAzureDoctorText(&out, report)
	output := out.String()
	for _, want := range []string{
		"Route: responses_streaming",
		"Route reason: deployment=corp-codex uses Responses API; catalog_model=gpt-5.3-codex supports Responses streaming",
		"Smoke response ID: resp_text",
		"Smoke usage: input=10 cached=3 output=4 reasoning=2 cache_creation=1",
		"Smoke cost estimate: $0.00012345 USD",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}
}

func TestRenderAzureDoctorTextIncludesRequestPreview(t *testing.T) {
	report := azureprovider.DiagnosticReport{
		Provider:    "azure",
		Route:       azureprovider.DiagnosticRouteResponsesNonStreaming,
		RouteReason: "deployment=corp-gpt55-pro uses Responses API; catalog_model=gpt-5.5-pro disables Responses streaming",
		RequestPreview: &azureprovider.DiagnosticRequestPreview{
			Requests: []azureprovider.DiagnosticRequestPreviewRequest{{
				Name:    "text",
				Route:   azureprovider.DiagnosticRouteResponsesNonStreaming,
				Method:  "POST",
				URL:     "https://example.openai.azure.com/openai/v1/responses",
				Headers: map[string]string{"api-key": "<redacted>"},
				Body:    map[string]any{"model": "corp-gpt55-pro", "store": false},
			}},
		},
	}

	var out bytes.Buffer
	renderAzureDoctorText(&out, report)
	output := out.String()
	for _, want := range []string{
		"Request preview:",
		`"api-key": "<redacted>"`,
		`"model": "corp-gpt55-pro"`,
		`"store": false`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}
}

func TestRenderAzureDoctorTextIncludesCapabilities(t *testing.T) {
	report := azureprovider.DiagnosticReport{
		Provider: "azure",
		Route:    azureprovider.DiagnosticRouteResponsesStreaming,
		Capabilities: &azureprovider.DiagnosticCapabilities{
			Deployment:         "corp-codex-deployment",
			CatalogModel:       "gpt-5.3-codex",
			Route:              azureprovider.DiagnosticRouteResponsesStreaming,
			ResponsesAPI:       true,
			ResponsesStreaming: true,
			Retention: azureprovider.DiagnosticRetentionCapability{
				PreviousResponseID: true,
			},
			ServerCompaction: azureprovider.DiagnosticServerCompactionCapability{
				RequestPayload:   true,
				CompactThreshold: 272000,
			},
		},
	}

	var out bytes.Buffer
	renderAzureDoctorText(&out, report)
	output := out.String()
	for _, want := range []string{
		"Capabilities:",
		`"responses_api": true`,
		`"previous_response_id": true`,
		`"compact_threshold": 272000`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}
}

func TestRenderAzureDoctorTextIncludesRetentionSmokeRequests(t *testing.T) {
	report := azureprovider.DiagnosticReport{
		Provider: "azure",
		Smoke: &azureprovider.DiagnosticSmokeResult{
			Ran:              true,
			RetentionPayload: true,
			UsageObserved:    true,
			Requests: []azureprovider.DiagnosticSmokeRequestResult{
				{
					Name:             "retention_initial",
					Ran:              true,
					RetentionPayload: true,
					ResponseID:       "resp_retention_initial",
					Duration:         "1ms",
					UsageObserved:    true,
				},
				{
					Name:               "retention_followup",
					Ran:                true,
					RetentionPayload:   true,
					ResponseID:         "resp_retention_followup",
					PreviousResponseID: "resp_retention_initial",
					Duration:           "2ms",
					UsageObserved:      true,
				},
			},
		},
	}

	var out bytes.Buffer
	renderAzureDoctorText(&out, report)
	output := out.String()
	for _, want := range []string{
		"Smoke request retention_initial: ok duration=1ms response_id=resp_retention_initial previous_response_id=(not returned)",
		"Smoke request retention_followup: ok duration=2ms response_id=resp_retention_followup previous_response_id=resp_retention_initial",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}
}

func TestRenderAzureDoctorJSONIncludesSmokeObservability(t *testing.T) {
	report := azureprovider.DiagnosticReport{
		Provider: "azure",
		Smoke: &azureprovider.DiagnosticSmokeResult{
			Ran:              true,
			ResponseID:       "resp_json",
			Duration:         "1ms",
			RetentionPayload: true,
			UsageObserved:    true,
			Usage: azureprovider.DiagnosticSmokeUsage{
				InputTokens:         10,
				OutputTokens:        4,
				ThinkingTokens:      2,
				CachedInputTokens:   3,
				CacheCreationTokens: 1,
			},
			Cost: azureprovider.DiagnosticSmokeCost{
				USD:                0.00012345,
				PricingUnavailable: false,
			},
			Requests: []azureprovider.DiagnosticSmokeRequestResult{{
				Name:               "retention_followup",
				Ran:                true,
				RetentionPayload:   true,
				ResponseID:         "resp_retention_followup",
				PreviousResponseID: "resp_json",
				UsageObserved:      true,
			}},
		},
	}

	var out bytes.Buffer
	if err := renderAzureDoctorJSON(&out, report); err != nil {
		t.Fatalf("renderAzureDoctorJSON() error = %v", err)
	}

	var got struct {
		Smoke struct {
			ResponseID       string `json:"response_id"`
			RetentionPayload bool   `json:"retention_payload"`
			UsageObserved    bool   `json:"usage_observed"`
			Usage            struct {
				InputTokens         int `json:"input_tokens"`
				OutputTokens        int `json:"output_tokens"`
				ThinkingTokens      int `json:"thinking_tokens"`
				CachedInputTokens   int `json:"cached_input_tokens"`
				CacheCreationTokens int `json:"cache_creation_tokens"`
			} `json:"usage"`
			Cost struct {
				USD                float64 `json:"usd"`
				PricingUnavailable bool    `json:"pricing_unavailable"`
			} `json:"cost"`
			Requests []struct {
				Name               string `json:"name"`
				Ran                bool   `json:"ran"`
				RetentionPayload   bool   `json:"retention_payload"`
				ResponseID         string `json:"response_id"`
				PreviousResponseID string `json:"previous_response_id"`
			} `json:"requests"`
		} `json:"smoke"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, out.String())
	}
	if got.Smoke.ResponseID != "resp_json" || !got.Smoke.UsageObserved {
		t.Fatalf("smoke metadata = %#v, want response_id and usage_observed", got.Smoke)
	}
	if got.Smoke.Usage.InputTokens != 10 ||
		got.Smoke.Usage.OutputTokens != 4 ||
		got.Smoke.Usage.ThinkingTokens != 2 ||
		got.Smoke.Usage.CachedInputTokens != 3 ||
		got.Smoke.Usage.CacheCreationTokens != 1 {
		t.Fatalf("smoke usage = %+v, want nested usage fields", got.Smoke.Usage)
	}
	if got.Smoke.Cost.USD != 0.00012345 || got.Smoke.Cost.PricingUnavailable {
		t.Fatalf("smoke cost = %+v, want nested cost fields", got.Smoke.Cost)
	}
	if !got.Smoke.RetentionPayload || len(got.Smoke.Requests) != 1 || got.Smoke.Requests[0].PreviousResponseID != "resp_json" {
		t.Fatalf("smoke retention JSON = %+v, want retention request metadata", got.Smoke)
	}
}
