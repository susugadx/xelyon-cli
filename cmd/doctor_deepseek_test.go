package cmd

import (
	"bytes"
	"strings"
	"testing"

	deepseekprovider "github.com/susugadx/xelyon-cli/internal/api/providers/deepseek"
)

func TestRunDeepSeekDoctorInvocation_JSONReportsExplicitModelCatalogAndThinking(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("DEEPSEEK_API_KEY", "sk-test")
	t.Setenv("DEEPSEEK_API_URL", "")

	cmd, out := newDoctorSubcommandTest(t, newDeepSeekDoctorCommand)

	doctorDeepSeekModelFlag = "corp-deepseek-model"
	doctorCatalogModelFlag = "deepseek-v4-flash"
	doctorJSONFlag = true

	if err := runDeepSeekDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runDeepSeekDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	if report.Provider != "deepseek" {
		t.Fatalf("provider = %q, want deepseek", report.Provider)
	}
	if report.Model != "corp-deepseek-model" || report.ModelSource != "--model" {
		t.Fatalf("model = %q (%s), want explicit model", report.Model, report.ModelSource)
	}
	if report.APIModel != "corp-deepseek-model" {
		t.Fatalf("api_model = %q, want request alias", report.APIModel)
	}
	if report.CatalogModel != "deepseek-v4-flash" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit catalog model", report.CatalogModel, report.CatalogModelSource)
	}
	if report.Route != "chat_completions" {
		t.Fatalf("route = %q, want chat_completions", report.Route)
	}
	if !report.ThinkingSupported || report.ThinkingType != "disabled" {
		t.Fatalf("thinking = supported:%t type:%q, want disabled V4 thinking payload", report.ThinkingSupported, report.ThinkingType)
	}
	catalogPolicy := requireDoctorJSONCheck(t, report.Checks, "catalog_policy")
	requireDoctorJSONCheckStatus(t, catalogPolicy, "ok")
	requireDoctorJSONCheckDetailContains(t, catalogPolicy, "max_output_tokens=384000")
	thinking := requireDoctorJSONCheck(t, report.Checks, "thinking")
	requireDoctorJSONCheckStatus(t, thinking, "ok")
	requireDoctorJSONCheckDetailContains(t, thinking, "thinking.type=disabled")
}

func TestRunDeepSeekDoctorInvocation_PrintRequestJSONDoesNotRequireAPIKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("DEEPSEEK_API_KEY", "")
	t.Setenv("DEEPSEEK_API_URL", "")

	cmd, out := newDoctorSubcommandTest(t, newDeepSeekDoctorCommand)

	doctorDeepSeekModelFlag = "deepseek-chat"
	doctorCatalogModelFlag = "deepseek-chat"
	doctorToolSmokeFlag = true
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runDeepSeekDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runDeepSeekDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	requireDoctorJSONPrintRequestOmittedSmoke(t, report.Smoke)
	requireDoctorJSONPrintRequestSkippedAuth(t, report.Checks)
	requireDoctorJSONRequestPreviewCount(t, report.RequestPreview, 1)
	request := requireDoctorJSONRequestPreviewAt(t, report.RequestPreview, 0, "tool")
	if request.Name != "tool" || !request.ToolPayload {
		t.Fatalf("preview request = %#v, want tool payload", request)
	}
	requireDoctorJSONRequestPreviewHeader(t, request, "Authorization", "Bearer <redacted>")
	body := requireDoctorJSONRequestPreviewBody[struct {
		Model      string         `json:"model"`
		MaxTokens  int            `json:"max_tokens"`
		Tools      []any          `json:"tools"`
		ToolChoice any            `json:"tool_choice"`
		Thinking   map[string]any `json:"thinking"`
	}](t, request)
	if body.Model != "deepseek-v4-flash" || body.MaxTokens != 64 || len(body.Tools) != 1 || body.ToolChoice == nil {
		t.Fatalf("preview body = %#v, want diagnostic tool body", body)
	}
	if body.Thinking["type"] != "disabled" {
		t.Fatalf("preview thinking = %#v, want disabled", body.Thinking)
	}
}

func TestRunDeepSeekDoctorInvocation_PrintRequestJSONReportsOpenAICompatibleProxyEndpointWarning(t *testing.T) {
	proxyURL := "https://deepseek.example/v1/chat/completions"

	t.Setenv("HOME", t.TempDir())
	t.Setenv("DEEPSEEK_API_KEY", "")
	t.Setenv("DEEPSEEK_API_URL", proxyURL)

	cmd, out := newDoctorSubcommandTest(t, newDeepSeekDoctorCommand)

	doctorDeepSeekModelFlag = "deepseek-v4-flash"
	doctorCatalogModelFlag = "deepseek-v4-flash"
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runDeepSeekDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runDeepSeekDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	if report.APIURL != proxyURL {
		t.Fatalf("api_url = %q, want configured proxy URL", report.APIURL)
	}
	requireDoctorJSONProxyWarning(t, report.Checks, "endpoint", "", proxyURL)
	requireDoctorJSONRequestPreviewURLs(t, report.RequestPreview, 1, proxyURL)
}

func TestRootCommand_DeepSeekDoctorCommandParsesFlags(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("DEEPSEEK_API_KEY", "sk-test")
	t.Setenv("DEEPSEEK_API_URL", "")

	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "deepseek", "--model", "corp-deepseek-model", "--catalog-model", "deepseek-v4-flash", "--json"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"model": "corp-deepseek-model"`) {
		t.Fatalf("output = %q, want parsed DeepSeek model", out.String())
	}
	if !strings.Contains(out.String(), `"catalog_model": "deepseek-v4-flash"`) {
		t.Fatalf("output = %q, want parsed DeepSeek catalog model", out.String())
	}
}

func TestRootCommand_DeepSeekDoctorHelpShowsMinimalFlags(t *testing.T) {
	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "deepseek", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	for _, want := range []string{"--model", "--catalog-model", "--smoke", "--tool-smoke", "--print-request", "--timeout", "--json", "Diagnose DeepSeek provider configuration", "exact Chat Completions endpoint", "/chat/completions"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output = %q, want DeepSeek doctor help substring %q", out.String(), want)
		}
	}
	for _, unwanted := range []string{"--capabilities", "--require-capability", "--retention-smoke", "--image-smoke", "--web-search-smoke", "--thinking-smoke", "--print-config"} {
		if strings.Contains(out.String(), unwanted) {
			t.Fatalf("output = %q, should not contain %s", out.String(), unwanted)
		}
	}
}

func TestRenderDeepSeekDoctorTextIncludesThinkingRequestPreviewAndSmokeObservability(t *testing.T) {
	report := deepseekprovider.DiagnosticReport{
		Provider:           "deepseek",
		APIURL:             "https://api.deepseek.com/chat/completions",
		Model:              "deepseek-v4-flash",
		ModelSource:        "test",
		APIModel:           "deepseek-v4-flash",
		CatalogModel:       "deepseek-v4-flash",
		CatalogModelSource: "test",
		Route:              deepseekprovider.DiagnosticRouteChatCompletions,
		RouteReason:        "DeepSeek provider uses OpenAI-compatible Chat Completions",
		ThinkingSupported:  true,
		ThinkingEnabled:    true,
		ThinkingType:       "enabled",
		ReasoningEffort:    "max",
		Checks: []deepseekprovider.DiagnosticCheck{
			{Name: "smoke", Status: deepseekprovider.DiagnosticStatusOK, Message: "live DeepSeek smoke request succeeded"},
		},
		RequestPreview: &deepseekprovider.DiagnosticRequestPreview{
			Requests: []deepseekprovider.DiagnosticRequestPreviewRequest{{
				Name:    "text",
				Route:   deepseekprovider.DiagnosticRouteChatCompletions,
				Method:  "POST",
				URL:     "https://api.deepseek.com/chat/completions",
				Headers: map[string]string{"Authorization": "Bearer <redacted>"},
				Body:    map[string]any{"model": "deepseek-v4-flash", "max_tokens": 64, "thinking": map[string]any{"type": "enabled"}},
			}},
		},
		Smoke: &deepseekprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         deepseekprovider.DiagnosticRouteChatCompletions,
			Content:       "xelyon deepseek doctor ok",
			Duration:      "1ms",
			UsageObserved: true,
			Usage: deepseekprovider.DiagnosticSmokeUsage{
				InputTokens:         10,
				OutputTokens:        4,
				ThinkingTokens:      0,
				CachedInputTokens:   3,
				CacheCreationTokens: 1,
			},
			Cost: deepseekprovider.DiagnosticSmokeCost{
				USD: 0.00012345,
			},
		},
	}

	var out bytes.Buffer
	renderDeepSeekDoctorText(&out, report)
	output := out.String()
	for _, want := range []string{
		"API model: deepseek-v4-flash",
		"Thinking: supported=true enabled=true type=enabled reasoning_effort=max",
		"Route reason: DeepSeek provider uses OpenAI-compatible Chat Completions",
		"Request preview:",
		`"Authorization": "Bearer <redacted>"`,
		"Smoke route: chat_completions",
		"Smoke usage: input=10 cached=3 output=4 reasoning=0 cache_creation=1",
		"Smoke cost estimate: $0.00012345 USD",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}
}
