package cmd

import (
	"bytes"
	"strings"
	"testing"

	bedrockprovider "github.com/susugadx/xelyon-cli/internal/api/providers/bedrock"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

const bedrockDoctorCatalogModelForTest = "global.anthropic.claude-sonnet-4-6"

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
	for _, want := range []string{"--model", "--catalog-model", "--tool-smoke", "--image-smoke", "--thinking-smoke", "--capabilities", "--require-capability", "--print-request", "Diagnose AWS Bedrock configuration"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output = %q, want Bedrock doctor help substring %q", out.String(), want)
		}
	}
}

func TestRootCommand_BedrockCapabilitiesRejectCrossProviderCatalogMetadata(t *testing.T) {
	setBedrockDoctorCommandTestEnv(t)

	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{
		"doctor", "bedrock",
		"--model", "amazon.nova-pro-v1:0",
		"--catalog-model", "gpt-5.4",
		"--capabilities",
		"--json",
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[struct {
		Capabilities struct {
			CatalogModel          string `json:"catalog_model"`
			ContextWindowTokens   int    `json:"context_window_tokens"`
			ContextWindowKnown    bool   `json:"context_window_known"`
			MaxOutputTokens       int    `json:"max_output_tokens"`
			MaxOutputTokensKnown  bool   `json:"max_output_tokens_known"`
			MaxOutputTokensSource string `json:"max_output_tokens_source"`
			Pricing               struct {
				Available bool   `json:"available"`
				Detail    string `json:"detail"`
			} `json:"pricing"`
		} `json:"capabilities"`
		Checks []doctorJSONCheck `json:"checks"`
	}](t, out)

	if report.Capabilities.CatalogModel != "gpt-5.4" {
		t.Fatalf("capabilities.catalog_model = %q, want explicit catalog model", report.Capabilities.CatalogModel)
	}
	if report.Capabilities.ContextWindowKnown || report.Capabilities.ContextWindowTokens != 0 {
		t.Fatalf("context window = %d known=%t, want unknown for non-Bedrock catalog", report.Capabilities.ContextWindowTokens, report.Capabilities.ContextWindowKnown)
	}
	if !report.Capabilities.MaxOutputTokensKnown ||
		report.Capabilities.MaxOutputTokens != 5000 ||
		report.Capabilities.MaxOutputTokensSource != providerdiag.MaxOutputSourceCatalog {
		t.Fatalf("max output = %d known=%t source=%q, want Bedrock request-model catalog fallback", report.Capabilities.MaxOutputTokens, report.Capabilities.MaxOutputTokensKnown, report.Capabilities.MaxOutputTokensSource)
	}
	if report.Capabilities.Pricing.Available || report.Capabilities.Pricing.Detail != "pricing=unavailable" {
		t.Fatalf("pricing = %+v, want unavailable for non-Bedrock catalog", report.Capabilities.Pricing)
	}

	catalogPolicy := requireDoctorJSONCheck(t, report.Checks, "catalog_policy")
	requireDoctorJSONCheckStatus(t, catalogPolicy, "warn")
	requireDoctorJSONCheckDetailContains(t, catalogPolicy, "catalog_model=gpt-5.4, context_window=unknown")
	for _, unwanted := range []string{"context_window=1000000", "max_output_tokens=64000", "pricing=input $2.50/M"} {
		if strings.Contains(catalogPolicy.Detail, unwanted) {
			t.Fatalf("catalog_policy detail = %q, should not contain cross-provider metadata %q", catalogPolicy.Detail, unwanted)
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
