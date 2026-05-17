package cmd

import (
	"bytes"
	"strings"
	"testing"

	claudeprovider "github.com/susugadx/xelyon-cli/internal/api/providers/claude"
)

func TestRunClaudeDoctorInvocation_JSONReportsExplicitModelCatalogAndCapabilities(t *testing.T) {
	setClaudeDoctorCommandTestEnv(t, "claude-key")

	cmd, out := newDoctorSubcommandTest(t, newClaudeDoctorCommand)

	doctorClaudeModelFlag = "corp-claude-model"
	doctorCatalogModelFlag = "claude-sonnet-4-6"
	doctorJSONFlag = true

	if err := runClaudeDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runClaudeDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	if report.Provider != "claude" {
		t.Fatalf("provider = %q, want claude", report.Provider)
	}
	if report.Model != "corp-claude-model" || report.ModelSource != "--model" {
		t.Fatalf("model = %q (%s), want explicit model", report.Model, report.ModelSource)
	}
	if report.CatalogModel != "claude-sonnet-4-6" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit catalog model", report.CatalogModel, report.CatalogModelSource)
	}
	if report.Route != "claude_messages" {
		t.Fatalf("route = %q, want claude_messages", report.Route)
	}
	if !report.FunctionCallingEnabled || !report.ImageInputSupported || !report.WebSearchSupported || !report.ContextManagementEnabled || !report.ClaudeCompactionSupported {
		t.Fatalf("capabilities = fc:%t image:%t web:%t context:%t compaction:%t, want all enabled",
			report.FunctionCallingEnabled,
			report.ImageInputSupported,
			report.WebSearchSupported,
			report.ContextManagementEnabled,
			report.ClaudeCompactionSupported,
		)
	}
	if report.AnthropicVersion != "2023-06-01" {
		t.Fatalf("anthropic_version = %q, want default", report.AnthropicVersion)
	}
	catalogPolicy := requireDoctorJSONCheck(t, report.Checks, "catalog_policy")
	requireDoctorJSONCheckStatus(t, catalogPolicy, "ok")
	requireDoctorJSONCheckDetailContains(t, catalogPolicy, "max_output_tokens=64000")
}

func TestRunClaudeDoctorInvocation_PrintRequestJSONDoesNotRequireAPIKey(t *testing.T) {
	setClaudeDoctorCommandTestEnv(t, "")

	cmd, out := newDoctorSubcommandTest(t, newClaudeDoctorCommand)

	doctorClaudeModelFlag = "corp-claude-model"
	doctorCatalogModelFlag = "claude-sonnet-4-6"
	doctorToolSmokeFlag = true
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runClaudeDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runClaudeDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	requireDoctorJSONPrintRequestOmittedSmoke(t, report.Smoke)
	requireDoctorJSONPrintRequestSkippedAuth(t, report.Checks)
	requireDoctorJSONRequestPreviewCount(t, report.RequestPreview, 1)
	request := requireDoctorJSONRequestPreviewAt(t, report.RequestPreview, 0, "tool")
	if request.Name != "tool" || !request.ToolPayload || request.Route != "claude_messages" {
		t.Fatalf("preview request = %#v, want Claude tool Messages request", request)
	}
	requireDoctorJSONRequestPreviewHeader(t, request, "x-api-key", "<redacted>")
	requireDoctorJSONRequestPreviewHeader(t, request, "anthropic-version", "2023-06-01")
	body := requireDoctorJSONRequestPreviewBody[struct {
		Model string `json:"model"`
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
		ToolChoice struct {
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"tool_choice"`
	}](t, request)
	if body.Model != "corp-claude-model" || len(body.Tools) != 1 || body.Tools[0].Name != "xelyon_claude_doctor_probe" {
		t.Fatalf("request body = %#v, want model and diagnostic Claude tool", body)
	}
	if body.ToolChoice.Type != "tool" || body.ToolChoice.Name != "xelyon_claude_doctor_probe" {
		t.Fatalf("tool_choice = %#v, want forced diagnostic Claude tool", body.ToolChoice)
	}
}

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

func TestRootCommand_ClaudeDoctorCommandParsesFlags(t *testing.T) {
	setClaudeDoctorCommandTestEnv(t, "claude-key")

	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{
		"doctor", "claude",
		"--model", "corp-claude-model",
		"--catalog-model", "claude-sonnet-4-6",
		"--smoke",
		"--tool-smoke",
		"--image-smoke",
		"--thinking-smoke",
		"--web-search-smoke",
		"--print-request",
		"--timeout", "1s",
		"--json",
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"model": "corp-claude-model"`) {
		t.Fatalf("output = %q, want parsed Claude model", out.String())
	}
	if !strings.Contains(out.String(), `"catalog_model": "claude-sonnet-4-6"`) {
		t.Fatalf("output = %q, want parsed Claude catalog model", out.String())
	}
	if !strings.Contains(out.String(), `"thinking_payload": true`) || !strings.Contains(out.String(), `"web_search_payload": true`) {
		t.Fatalf("output = %q, want parsed Claude specialized smoke flags", out.String())
	}
}

func TestRootCommand_ClaudeDoctorHelpShowsMinimalFlags(t *testing.T) {
	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "claude", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	for _, want := range []string{"--model", "--catalog-model", "--smoke", "--tool-smoke", "--image-smoke", "--thinking-smoke", "--web-search-smoke", "--print-request", "--timeout", "--json", "Diagnose Claude provider configuration", "exact Messages", "/v1/messages"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output = %q, want Claude doctor help substring %q", out.String(), want)
		}
	}
	for _, unwanted := range []string{"--capabilities", "--require-capability", "--retention-smoke", "--print-config"} {
		if strings.Contains(out.String(), unwanted) {
			t.Fatalf("output = %q, should not contain %s", out.String(), unwanted)
		}
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
	output := out.String()
	for _, want := range []string{
		"Claude doctor",
		"Route: claude_messages",
		"Capabilities: function_calling=true image_input=true web_search=true thinking=true context_management=true claude_compaction=true thinking_type=adaptive",
		"Anthropic beta: context-management-2025-06-27",
		"Request preview:",
		`"x-api-key": "<redacted>"`,
		"Smoke route: claude_messages",
		"Smoke usage: input=10 cached=3 output=4 reasoning=0 cache_creation=1",
		"Smoke cost estimate: $0.00012345 USD",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}
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
	for _, want := range []string{
		"Smoke cost estimate text: $0.00012345 USD",
		"Smoke cost estimate web_search: N/A (usage unavailable)",
		"Smoke total cost estimate: N/A (usage unavailable)",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("renderClaudeDoctorText() output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "Smoke total cost estimate: $") {
		t.Fatalf("renderClaudeDoctorText() printed partial total cost:\n%s", output)
	}
}

func setClaudeDoctorCommandTestEnv(t *testing.T, apiKey string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ANTHROPIC_API_KEY", apiKey)
	t.Setenv("ANTHROPIC_API_URL", "")
	t.Setenv("CLAUDE_FUNCTION_CALLING", "")
	t.Setenv("XELYON_MODEL", "")
}
