package clidoctor

import (
	"bytes"
	"testing"

	bedrockprovider "github.com/susugadx/xelyon-cli/internal/api/providers/bedrock"
)

const bedrockDoctorCatalogModelForTest = "global.anthropic.claude-sonnet-4-6"

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
