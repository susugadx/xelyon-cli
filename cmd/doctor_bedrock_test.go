package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	bedrockprovider "github.com/susugadx/xelyon-cli/internal/api/providers/bedrock"
)

const bedrockDoctorCatalogModelForTest = "global.anthropic.claude-sonnet-4-6"

func TestRunBedrockDoctorInvocation_JSONReportsExplicitModel(t *testing.T) {
	setBedrockDoctorCommandTestEnv(t)

	cmd, out := newDoctorSubcommandTest(t, newBedrockDoctorCommand)

	doctorBedrockModelFlag = "corp-bedrock-sonnet"
	doctorCatalogModelFlag = "global.anthropic.claude-sonnet-4-6"
	doctorJSONFlag = true

	if err := runBedrockDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runBedrockDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	var report struct {
		Provider     string `json:"provider"`
		Region       string `json:"region"`
		Model        string `json:"model"`
		CatalogModel string `json:"catalog_model"`
		Route        string `json:"route"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, out.String())
	}
	if report.Provider != "bedrock" {
		t.Fatalf("provider = %q, want bedrock", report.Provider)
	}
	if report.Region != "us-east-1" {
		t.Fatalf("region = %q, want us-east-1", report.Region)
	}
	if report.Model != "corp-bedrock-sonnet" || report.CatalogModel != "global.anthropic.claude-sonnet-4-6" {
		t.Fatalf("model/catalog = %q/%q, want explicit values", report.Model, report.CatalogModel)
	}
	if report.Route != "claude_messages" {
		t.Fatalf("route = %q, want claude_messages", report.Route)
	}
}

func TestRootCommand_BedrockDoctorCommandParsesFlags(t *testing.T) {
	setBedrockDoctorCommandTestEnv(t)

	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "bedrock", "--model", "amazon.nova-pro-v1:0", "--catalog-model", "amazon.nova-pro-v1:0", "--print-request", "--json"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"model": "amazon.nova-pro-v1:0"`) {
		t.Fatalf("output = %q, want parsed Bedrock model", out.String())
	}
}

func TestRootCommand_BedrockDoctorHelpShowsDoctorFlags(t *testing.T) {
	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "bedrock", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	for _, want := range []string{"--model", "--catalog-model", "--tool-smoke", "--image-smoke", "--thinking-smoke", "--print-request", "Diagnose AWS Bedrock configuration"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output = %q, want Bedrock doctor help substring %q", out.String(), want)
		}
	}
}

func TestRunBedrockDoctorInvocation_PrintRequestJSONDoesNotRequireAWSCredentials(t *testing.T) {
	setBedrockDoctorCommandTestEnv(t)
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SESSION_TOKEN", "")

	cmd, out := newDoctorSubcommandTest(t, newBedrockDoctorCommand)

	doctorBedrockModelFlag = "corp-bedrock-sonnet"
	doctorCatalogModelFlag = bedrockDoctorCatalogModelForTest
	doctorToolSmokeFlag = true
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runBedrockDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runBedrockDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[struct {
		Smoke          any `json:"smoke"`
		RequestPreview struct {
			Requests []struct {
				Name        string            `json:"name"`
				ToolPayload bool              `json:"tool_payload"`
				Route       string            `json:"route"`
				Operation   string            `json:"operation"`
				ModelID     string            `json:"model_id"`
				URL         string            `json:"url"`
				Headers     map[string]string `json:"headers"`
				Body        struct {
					AnthropicVersion string `json:"anthropic_version"`
					Tools            []struct {
						Name string `json:"name"`
					} `json:"tools"`
				} `json:"body"`
			} `json:"requests"`
		} `json:"request_preview"`
		Checks []doctorJSONCheck `json:"checks"`
	}](t, out)
	if report.Smoke != nil {
		t.Fatalf("smoke = %#v, want omitted for --print-request", report.Smoke)
	}
	requireNoDoctorJSONChecks(t, report.Checks, "auth")
	if len(report.RequestPreview.Requests) != 1 {
		t.Fatalf("request_preview = %#v, want one tool request", report.RequestPreview)
	}
	request := report.RequestPreview.Requests[0]
	if request.Name != "tool" || !request.ToolPayload || request.Route != "claude_messages" || request.Operation != "invoke_model_with_response_stream" {
		t.Fatalf("preview request = %#v, want Bedrock Claude tool invoke request", request)
	}
	if request.ModelID != "corp-bedrock-sonnet" || !strings.Contains(request.URL, "/invoke-with-response-stream") {
		t.Fatalf("preview target = model_id:%q url:%q, want Bedrock invoke target", request.ModelID, request.URL)
	}
	if request.Headers["Authorization"] != "<redacted: AWS SigV4>" {
		t.Fatalf("Authorization preview = %q, want redacted SigV4", request.Headers["Authorization"])
	}
	if request.Body.AnthropicVersion == "" || len(request.Body.Tools) != 1 || request.Body.Tools[0].Name != "xelyon_bedrock_doctor_probe" {
		t.Fatalf("request body = %#v, want Bedrock Claude body with diagnostic tool", request.Body)
	}
}

func TestRenderBedrockDoctorTextIncludesSmokeRequests(t *testing.T) {
	report := bedrockprovider.DiagnosticReport{
		Provider:           "bedrock",
		Region:             "us-east-1",
		Model:              "amazon.nova-pro-v1:0",
		ModelSource:        "test",
		CatalogModel:       "amazon.nova-pro-v1:0",
		CatalogModelSource: "test",
		Route:              "converse_stream",
		Checks: []bedrockprovider.DiagnosticCheck{
			{Name: "smoke", Status: bedrockprovider.DiagnosticStatusOK, Message: "live Bedrock smoke requests completed"},
		},
		Smoke: &bedrockprovider.DiagnosticSmokeResult{
			Ran:           true,
			UsageObserved: true,
			Usage: bedrockprovider.DiagnosticSmokeUsage{
				InputTokens:         10,
				CachedInputTokens:   2,
				OutputTokens:        4,
				ThinkingTokens:      1,
				CacheCreationTokens: 3,
			},
			Cost: bedrockprovider.DiagnosticSmokeCost{USD: 0.00012345},
			Requests: []bedrockprovider.DiagnosticSmokeRequestResult{
				{
					Name:          "text",
					Ran:           true,
					RequestID:     "req_text",
					Duration:      "1ms",
					Content:       "ok",
					UsageObserved: true,
					Usage: bedrockprovider.DiagnosticSmokeUsage{
						InputTokens:         10,
						CachedInputTokens:   2,
						OutputTokens:        4,
						ThinkingTokens:      1,
						CacheCreationTokens: 3,
					},
					Cost: bedrockprovider.DiagnosticSmokeCost{USD: 0.00012345},
				},
				{Name: "image", Skipped: true, SkipReason: "unsupported route"},
			},
		},
	}

	var out bytes.Buffer
	renderBedrockDoctorText(&out, report)
	output := out.String()
	for _, want := range []string{
		"Smoke request text: ok duration=1ms request_id=req_text",
		"Smoke usage text: input=10 cached=2 output=4 reasoning=1 cache_creation=3",
		"Smoke cost estimate text: $0.00012345 USD",
		"Smoke request image: skipped (unsupported route)",
		"Smoke total usage: input=10 cached=2 output=4 reasoning=1 cache_creation=3",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}
}

func TestRenderBedrockDoctorTextIncludesRequestPreview(t *testing.T) {
	report := bedrockprovider.DiagnosticReport{
		Provider:           "bedrock",
		Region:             "us-east-1",
		Model:              "global.anthropic.claude-sonnet-4-6",
		ModelSource:        "test",
		CatalogModel:       "global.anthropic.claude-sonnet-4-6",
		CatalogModelSource: "test",
		Route:              "claude_messages",
		Checks: []bedrockprovider.DiagnosticCheck{
			{Name: "request_preview", Status: bedrockprovider.DiagnosticStatusOK, Message: "Bedrock request preview was built without sending a live request"},
		},
		RequestPreview: &bedrockprovider.DiagnosticRequestPreview{
			Requests: []bedrockprovider.DiagnosticRequestPreviewRequest{{
				Name:      "text",
				Route:     "claude_messages",
				Operation: "invoke_model_with_response_stream",
				ModelID:   "global.anthropic.claude-sonnet-4-6",
				Method:    "POST",
				URL:       "https://bedrock-runtime.us-east-1.amazonaws.com/model/global.anthropic.claude-sonnet-4-6/invoke-with-response-stream",
				Headers:   map[string]string{"Authorization": "<redacted: AWS SigV4>"},
				Body:      map[string]any{"anthropic_version": "bedrock-2023-05-31"},
			}},
		},
	}

	var out bytes.Buffer
	renderBedrockDoctorText(&out, report)
	output := out.String()
	for _, want := range []string{
		"Request preview:",
		`"operation": "invoke_model_with_response_stream"`,
		`"Authorization": "<redacted: AWS SigV4>"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}
}

func TestRenderBedrockDoctorTextMarksPartialTotalCostUnavailable(t *testing.T) {
	report := bedrockprovider.DiagnosticReport{
		Provider:           "bedrock",
		Region:             "us-east-1",
		Model:              "global.anthropic.claude-sonnet-4-6",
		ModelSource:        "test",
		CatalogModel:       "global.anthropic.claude-sonnet-4-6",
		CatalogModelSource: "test",
		Route:              "claude_messages",
		Smoke: &bedrockprovider.DiagnosticSmokeResult{
			Ran:           true,
			UsageObserved: false,
			Usage:         bedrockprovider.DiagnosticSmokeUsage{InputTokens: 10, OutputTokens: 4},
			Cost:          bedrockprovider.DiagnosticSmokeCost{USD: 0.00012345},
			Requests: []bedrockprovider.DiagnosticSmokeRequestResult{
				{
					Name:          "text",
					Ran:           true,
					RequestID:     "req_text",
					Duration:      "1ms",
					UsageObserved: true,
					Usage:         bedrockprovider.DiagnosticSmokeUsage{InputTokens: 10, OutputTokens: 4},
					Cost:          bedrockprovider.DiagnosticSmokeCost{USD: 0.00012345},
				},
				{
					Name:          "thinking",
					Ran:           true,
					RequestID:     "req_thinking",
					Duration:      "1ms",
					UsageObserved: false,
				},
			},
		},
	}

	var out bytes.Buffer
	renderBedrockDoctorText(&out, report)
	output := out.String()
	if !strings.Contains(output, "Smoke total cost estimate: N/A (usage unavailable)") {
		t.Fatalf("output = %q, want unavailable total cost for partial usage", output)
	}
	if strings.Contains(output, "Smoke total cost estimate: $") {
		t.Fatalf("output = %q, should not print a dollar total for partial usage", output)
	}
}

func TestRenderBedrockDoctorJSONUsesRequestIDOnly(t *testing.T) {
	report := bedrockprovider.DiagnosticReport{
		Provider: "bedrock",
		Smoke: &bedrockprovider.DiagnosticSmokeResult{
			Ran: true,
			Requests: []bedrockprovider.DiagnosticSmokeRequestResult{{
				Name:      "text",
				Ran:       true,
				RequestID: "req_json",
			}},
		},
	}

	var out bytes.Buffer
	if err := renderBedrockDoctorJSON(&out, report); err != nil {
		t.Fatalf("renderBedrockDoctorJSON() error = %v", err)
	}
	if strings.Contains(out.String(), "response_id") {
		t.Fatalf("output = %q, should not contain response_id alias", out.String())
	}
	var parsed struct {
		Smoke struct {
			Requests []struct {
				RequestID string `json:"request_id"`
			} `json:"requests"`
		} `json:"smoke"`
	}
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, out.String())
	}
	if len(parsed.Smoke.Requests) != 1 || parsed.Smoke.Requests[0].RequestID != "req_json" {
		t.Fatalf("requests = %#v, want request_id", parsed.Smoke.Requests)
	}
}

func setBedrockDoctorCommandTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("BEDROCK_FUNCTION_CALLING", "")
	t.Setenv("XELYON_MODEL", "")
}
