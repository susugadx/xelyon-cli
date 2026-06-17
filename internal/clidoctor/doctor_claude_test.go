package clidoctor

import (
	"bytes"
	"strings"
	"testing"

	claudeprovider "github.com/susugadx/xelyon-cli/internal/api/providers/claude"
)

func TestRunClaudeDoctorInvocation_PrintRequestJSONReportsProxyEndpointWarning(t *testing.T) {
	proxyURL := "https://claude.example/proxy"
	setClaudeDoctorCommandTestEnv(t, "")
	t.Setenv("ANTHROPIC_API_URL", proxyURL)

	cmd, out := newDoctorSubcommandTest(t, newClaudeDoctorCommand)

	doctorClaudeModelFlag = "corp-claude-model"
	doctorCatalogModelFlag = "claude-sonnet-4-6"
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runClaudeDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runClaudeDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	if report.APIURL != proxyURL {
		t.Fatalf("api_url = %q, want configured proxy URL", report.APIURL)
	}
	requireDoctorJSONProxyWarning(t, report.Checks, "endpoint", "", proxyURL)
	requireDoctorJSONRequestPreviewURLs(t, report.RequestPreview, 1, proxyURL)
}

func TestRunClaudeDoctorInvocation_FailsForMissingKey(t *testing.T) {
	setClaudeDoctorCommandTestEnv(t, "")

	cmd, out := newDoctorSubcommandTest(t, newClaudeDoctorCommand)

	err := runClaudeDoctorInvocation(cmd, nil)
	if err == nil {
		t.Fatalf("runClaudeDoctorInvocation() error = nil, want diagnostics failure\noutput:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "ANTHROPIC_API_KEY") {
		t.Fatalf("output = %q, want ANTHROPIC_API_KEY failure", out.String())
	}
}

func TestRenderClaudeDoctorTextIncludesRequestPreviewAndSmokeObservability(t *testing.T) {
	report := claudeprovider.DiagnosticReport{
		Provider:                  "claude",
		APIURL:                    "https://api.anthropic.com/v1/messages",
		Model:                     "claude-sonnet-4-6",
		ModelSource:               "test",
		CatalogModel:              "claude-sonnet-4-6",
		CatalogModelSource:        "test",
		Route:                     claudeprovider.DiagnosticRouteClaudeMessages,
		RouteReason:               "Claude text, tool, image, thinking, and native web search diagnostics use Anthropic Messages",
		FunctionCallingEnabled:    true,
		ImageInputSupported:       true,
		WebSearchSupported:        true,
		ThinkingEnabled:           true,
		ThinkingType:              "adaptive",
		ContextManagementEnabled:  true,
		ClaudeCompactionSupported: true,
		AnthropicVersion:          "2023-06-01",
		AnthropicBeta:             []string{"context-management-2025-06-27"},
		Checks: []claudeprovider.DiagnosticCheck{
			{Name: "smoke", Status: claudeprovider.DiagnosticStatusOK, Message: "live Claude smoke request succeeded"},
		},
		RequestPreview: &claudeprovider.DiagnosticRequestPreview{
			Requests: []claudeprovider.DiagnosticRequestPreviewRequest{{
				Name:    "text",
				Route:   claudeprovider.DiagnosticRouteClaudeMessages,
				Method:  "POST",
				URL:     "https://example.test/claude",
				Headers: map[string]string{"x-api-key": "<redacted>"},
				Body:    map[string]any{"model": "claude-sonnet-4-6"},
			}},
		},
		Smoke: &claudeprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         claudeprovider.DiagnosticRouteClaudeMessages,
			Content:       "xelyon claude doctor ok",
			Duration:      "1ms",
			UsageObserved: true,
			Usage: claudeprovider.DiagnosticSmokeUsage{
				InputTokens:         10,
				OutputTokens:        4,
				ThinkingTokens:      0,
				CachedInputTokens:   3,
				CacheCreationTokens: 1,
			},
			Cost: claudeprovider.DiagnosticSmokeCost{
				USD: 0.00012345,
			},
		},
	}

	var out bytes.Buffer
	renderClaudeDoctorText(&out, report)
	requireDoctorContractTextContainsAll(t, out.String(), []string{
		"Claude doctor",
		"Route: claude_messages",
		"Capabilities: function_calling=true image_input=true web_search=true thinking=true context_management=true claude_compaction=true thinking_type=adaptive",
		"Anthropic beta: context-management-2025-06-27",
		"Request preview:",
		`"x-api-key": "<redacted>"`,
		"Smoke route: claude_messages",
		"Smoke usage: input=10 cached=3 output=4 reasoning=0 cache_creation=1",
		"Smoke cost estimate: $0.00012345 USD",
	})
}

func TestRenderClaudeDoctorText_SkippedToolSmokeRequest(t *testing.T) {
	report := claudeprovider.DiagnosticReport{
		Provider:    "claude",
		Model:       "claude-sonnet-4-6",
		ModelSource: "test",
		Route:       claudeprovider.DiagnosticRouteClaudeMessages,
		APIURL:      "https://api.anthropic.com/v1/messages",
		Smoke: &claudeprovider.DiagnosticSmokeResult{
			Ran: true,
			Requests: []claudeprovider.DiagnosticSmokeRequestResult{{
				Name:        "tool",
				Skipped:     true,
				SkipReason:  "Claude function calling payloads are disabled (CLAUDE_FUNCTION_CALLING=0)",
				ToolPayload: true,
			}},
		},
	}

	var out bytes.Buffer
	renderClaudeDoctorText(&out, report)
	want := "Smoke request tool: skipped (Claude function calling payloads are disabled (CLAUDE_FUNCTION_CALLING=0))"
	if !strings.Contains(out.String(), want) {
		t.Fatalf("renderClaudeDoctorText() output missing %q:\n%s", want, out.String())
	}
}

func TestRenderClaudeDoctorText_MultiSmokePartialUsageDoesNotPrintTotalCost(t *testing.T) {
	report := claudeprovider.DiagnosticReport{
		Provider:    "claude",
		Model:       "claude-sonnet-4-6",
		ModelSource: "test",
		Route:       "mixed",
		APIURL:      "https://api.anthropic.com/v1/messages",
		Smoke: &claudeprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         "mixed",
			UsageObserved: false,
			Usage: claudeprovider.DiagnosticSmokeUsage{
				InputTokens:  10,
				OutputTokens: 4,
			},
			Cost: claudeprovider.DiagnosticSmokeCost{
				USD: 0.00012345,
			},
			Requests: []claudeprovider.DiagnosticSmokeRequestResult{
				{
					Name:          "text",
					Ran:           true,
					Route:         claudeprovider.DiagnosticRouteClaudeMessages,
					Duration:      "1ms",
					UsageObserved: true,
					Usage: claudeprovider.DiagnosticSmokeUsage{
						InputTokens:  10,
						OutputTokens: 4,
					},
					Cost: claudeprovider.DiagnosticSmokeCost{
						USD: 0.00012345,
					},
				},
				{
					Name:             "web_search",
					Ran:              true,
					Route:            claudeprovider.DiagnosticRouteClaudeWebSearch,
					Duration:         "1ms",
					WebSearchPayload: true,
					UsageObserved:    false,
				},
			},
		},
	}

	var out bytes.Buffer
	renderClaudeDoctorText(&out, report)
	output := out.String()
	requireDoctorContractTextContainsAll(t, output, []string{
		"Smoke cost estimate text: $0.00012345 USD",
		"Smoke cost estimate web_search: N/A (usage unavailable)",
	})
	requireDoctorSmokeTextUnavailableTotalCost(t, output)
}

func setClaudeDoctorCommandTestEnv(t *testing.T, apiKey string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ANTHROPIC_API_KEY", apiKey)
	t.Setenv("ANTHROPIC_API_URL", "")
	t.Setenv("CLAUDE_FUNCTION_CALLING", "")
	t.Setenv("XELYON_MODEL", "")
}
