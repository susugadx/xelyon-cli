package clidoctor

import (
	"bytes"
	"os"
	"strings"
	"testing"

	azureprovider "github.com/susugadx/xelyon-cli/internal/api/providers/azure"
)

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
	requireDoctorContractTextContainsAll(t, out.String(), []string{
		"Route: responses_streaming",
		"Route reason: deployment=corp-codex uses Responses API; catalog_model=gpt-5.3-codex supports Responses streaming",
		"Smoke response ID: resp_text",
		"Smoke usage: input=10 cached=3 output=4 reasoning=2 cache_creation=1",
		"Smoke cost estimate: $0.00012345 USD",
	})
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
	requireDoctorContractTextContainsAll(t, out.String(), []string{
		"Smoke request retention_initial: ok duration=1ms response_id=resp_retention_initial previous_response_id=(not returned)",
		"Smoke request retention_followup: ok duration=2ms response_id=resp_retention_followup previous_response_id=resp_retention_initial",
	})
}
