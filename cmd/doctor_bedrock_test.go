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
	requireDoctorSmokeTextUnavailableTotalCost(t, out.String())
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
