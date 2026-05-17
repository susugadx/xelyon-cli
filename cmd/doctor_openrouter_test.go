package cmd

import (
	"bytes"
	"strings"
	"testing"

	openrouterprovider "github.com/susugadx/xelyon-cli/internal/api/providers/openrouter"
)

func TestRunOpenRouterDoctorInvocation_JSONReportsExplicitModelAndCatalogModel(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENROUTER_API_KEY", "sk-or-test")
	t.Setenv("OPENROUTER_API_URL", "")

	cmd, out := newDoctorSubcommandTest(t, newOpenRouterDoctorCommand)

	doctorOpenRouterModelFlag = "corp-openrouter-model"
	doctorCatalogModelFlag = "openai/gpt-5.4"
	doctorJSONFlag = true

	if err := runOpenRouterDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runOpenRouterDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	if report.Provider != "openrouter" {
		t.Fatalf("provider = %q, want openrouter", report.Provider)
	}
	if report.Model != "corp-openrouter-model" || report.ModelSource != "--model" {
		t.Fatalf("model = %q (%s), want explicit model", report.Model, report.ModelSource)
	}
	if report.CatalogModel != "openai/gpt-5.4" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit catalog model", report.CatalogModel, report.CatalogModelSource)
	}
	if report.UpstreamProvider != "openai" || report.UpstreamModel != "gpt-5.4" {
		t.Fatalf("upstream = %s/%s, want openai/gpt-5.4", report.UpstreamProvider, report.UpstreamModel)
	}
	if report.Route != "chat_completions" {
		t.Fatalf("route = %q, want chat_completions", report.Route)
	}
	catalogPolicy := requireDoctorJSONCheck(t, report.Checks, "catalog_policy")
	requireDoctorJSONCheckStatus(t, catalogPolicy, "ok")
	requireDoctorJSONCheckDetailContains(t, catalogPolicy, "max_output_tokens=64000")
	imageInput := requireDoctorJSONCheck(t, report.Checks, "image_input")
	requireDoctorJSONCheckStatus(t, imageInput, "ok")
}

func TestRunOpenRouterDoctorInvocation_PrintRequestJSONDoesNotRequireAPIKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("OPENROUTER_API_URL", "")
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "1")

	cmd, out := newDoctorSubcommandTest(t, newOpenRouterDoctorCommand)

	doctorOpenRouterModelFlag = "anthropic/claude-sonnet-4.6"
	doctorCatalogModelFlag = "anthropic/claude-sonnet-4.6"
	doctorToolSmokeFlag = true
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runOpenRouterDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runOpenRouterDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	requireDoctorJSONPrintRequestOmittedSmoke(t, report.Smoke)
	requireDoctorJSONPrintRequestSkippedAuth(t, report.Checks)
	requireDoctorJSONRequestPreviewCount(t, report.RequestPreview, 1)
	request := requireDoctorJSONRequestPreviewAt(t, report.RequestPreview, 0, "tool")
	if request.Name != "tool" || !request.ToolPayload || request.Route != "anthropic_messages" {
		t.Fatalf("preview request = %#v, want Anthropic Skin tool payload", request)
	}
	requireDoctorJSONRequestPreviewHeader(t, request, "Authorization", "Bearer <redacted>")
	requireDoctorJSONRequestPreviewHeader(t, request, "X-Title", "XELYON CLI")
	body := requireDoctorJSONRequestPreviewBody[struct {
		Model            string `json:"model"`
		AnthropicVersion string `json:"anthropic_version"`
		MaxTokens        int    `json:"max_tokens"`
		Tools            []any  `json:"tools"`
		ToolChoice       struct {
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"tool_choice"`
		ContextManagement any `json:"context_management"`
	}](t, request)
	if body.Model != "anthropic/claude-sonnet-4.6" ||
		body.AnthropicVersion == "" ||
		body.MaxTokens != 64 ||
		len(body.Tools) != 1 ||
		body.ToolChoice.Type != "tool" ||
		body.ToolChoice.Name != "xelyon_openrouter_doctor_probe" ||
		body.ContextManagement == nil {
		t.Fatalf("preview body = %#v, want diagnostic Anthropic Skin body", body)
	}
}

func TestRunOpenRouterDoctorInvocation_PrintRequestJSONReportsMessagesEndpointFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("OPENROUTER_API_URL", "https://openrouter.example/v1/messages")
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "1")

	cmd, out := newDoctorSubcommandTest(t, newOpenRouterDoctorCommand)

	doctorOpenRouterModelFlag = "anthropic/claude-sonnet-4.6"
	doctorCatalogModelFlag = "anthropic/claude-sonnet-4.6"
	doctorToolSmokeFlag = true
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runOpenRouterDoctorInvocation(cmd, nil); err == nil {
		t.Fatalf("runOpenRouterDoctorInvocation() error = nil, want endpoint failure\noutput:\n%s", out.String())
	}

	report := unmarshalDoctorJSON[struct {
		Checks []doctorJSONCheck `json:"checks"`
	}](t, out)
	requireNoDoctorJSONChecks(t, report.Checks, "auth")
	endpoint := requireDoctorJSONCheck(t, report.Checks, "endpoint")
	requireDoctorJSONCheckStatus(t, endpoint, "fail")
	requireDoctorJSONCheckDetailContains(t, endpoint, "https://openrouter.example/v1/messages")
	requireDoctorJSONCheckSuggestionContains(t, endpoint, "/v1/chat/completions")
}

func TestRootCommand_OpenRouterDoctorCommandParsesFlags(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENROUTER_API_KEY", "sk-or-test")
	t.Setenv("OPENROUTER_API_URL", "")

	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "openrouter", "--model", "corp-openrouter-model", "--catalog-model", "openai/gpt-5.4", "--json"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"model": "corp-openrouter-model"`) {
		t.Fatalf("output = %q, want parsed OpenRouter model", out.String())
	}
	if !strings.Contains(out.String(), `"catalog_model": "openai/gpt-5.4"`) {
		t.Fatalf("output = %q, want parsed OpenRouter catalog model", out.String())
	}
}

func TestRootCommand_OpenRouterDoctorHelpShowsMinimalFlags(t *testing.T) {
	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "openrouter", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	for _, want := range []string{
		"--model",
		"--catalog-model",
		"--smoke",
		"--tool-smoke",
		"--print-request",
		"--timeout",
		"--json",
		"Diagnose OpenRouter provider configuration",
		"Chat Completions endpoint",
		"should not be configured directly",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output = %q, want OpenRouter doctor help substring %q", out.String(), want)
		}
	}
	for _, unwanted := range []string{"--capabilities", "--require-capability", "--retention-smoke", "--image-smoke", "--web-search-smoke", "--thinking-smoke", "--print-config"} {
		if strings.Contains(out.String(), unwanted) {
			t.Fatalf("output = %q, should not contain %s", out.String(), unwanted)
		}
	}
}

func TestRenderOpenRouterDoctorTextIncludesRequestPreviewAndSmokeObservability(t *testing.T) {
	report := openrouterprovider.DiagnosticReport{
		Provider:           "openrouter",
		APIURL:             "https://openrouter.ai/api/v1/messages",
		Model:              "anthropic/claude-sonnet-4.6",
		ModelSource:        "test",
		CatalogModel:       "anthropic/claude-sonnet-4.6",
		CatalogModelSource: "test",
		UpstreamProvider:   "anthropic",
		UpstreamModel:      "claude-sonnet-4.6",
		Route:              openrouterprovider.DiagnosticRouteAnthropicMessages,
		RouteReason:        "request model anthropic/claude-sonnet-4.6 enables OpenRouter Anthropic Skin context management",
		Checks: []openrouterprovider.DiagnosticCheck{
			{Name: "smoke", Status: openrouterprovider.DiagnosticStatusOK, Message: "live OpenRouter smoke request succeeded"},
		},
		RequestPreview: &openrouterprovider.DiagnosticRequestPreview{
			Requests: []openrouterprovider.DiagnosticRequestPreviewRequest{{
				Name:    "text",
				Route:   openrouterprovider.DiagnosticRouteAnthropicMessages,
				Method:  "POST",
				URL:     "https://openrouter.ai/api/v1/messages",
				Headers: map[string]string{"Authorization": "Bearer <redacted>", "X-Title": "XELYON CLI"},
				Body:    map[string]any{"model": "anthropic/claude-sonnet-4.6", "max_tokens": 64},
			}},
		},
		Smoke: &openrouterprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         openrouterprovider.DiagnosticRouteAnthropicMessages,
			Content:       "xelyon openrouter doctor ok",
			Duration:      "1ms",
			UsageObserved: true,
			Usage: openrouterprovider.DiagnosticSmokeUsage{
				InputTokens:         10,
				OutputTokens:        4,
				ThinkingTokens:      2,
				CachedInputTokens:   3,
				CacheCreationTokens: 1,
			},
			Cost: openrouterprovider.DiagnosticSmokeCost{
				USD: 0.00012345,
			},
		},
	}

	var out bytes.Buffer
	renderOpenRouterDoctorText(&out, report)
	output := out.String()
	for _, want := range []string{
		"Upstream model: anthropic/claude-sonnet-4.6",
		"Route reason: request model anthropic/claude-sonnet-4.6 enables OpenRouter Anthropic Skin context management",
		"Request preview:",
		`"Authorization": "Bearer <redacted>"`,
		"Smoke route: anthropic_messages",
		"Smoke usage: input=10 cached=3 output=4 reasoning=2 cache_creation=1",
		"Smoke cost estimate: $0.00012345 USD",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}
}
