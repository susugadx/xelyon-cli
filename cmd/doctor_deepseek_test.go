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

	report := unmarshalDoctorJSON[struct {
		Provider           string            `json:"provider"`
		Model              string            `json:"model"`
		ModelSource        string            `json:"model_source"`
		APIModel           string            `json:"api_model"`
		CatalogModel       string            `json:"catalog_model"`
		CatalogModelSource string            `json:"catalog_model_source"`
		Route              string            `json:"route"`
		ThinkingSupported  bool              `json:"thinking_supported"`
		ThinkingType       string            `json:"thinking_type"`
		Checks             []doctorJSONCheck `json:"checks"`
	}](t, out)
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

	report := unmarshalDoctorJSON[struct {
		Smoke          any `json:"smoke"`
		RequestPreview struct {
			Requests []struct {
				Name        string            `json:"name"`
				ToolPayload bool              `json:"tool_payload"`
				URL         string            `json:"url"`
				Headers     map[string]string `json:"headers"`
				Body        struct {
					Model      string         `json:"model"`
					MaxTokens  int            `json:"max_tokens"`
					Tools      []any          `json:"tools"`
					ToolChoice any            `json:"tool_choice"`
					Thinking   map[string]any `json:"thinking"`
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
	if request.Name != "tool" || !request.ToolPayload {
		t.Fatalf("preview request = %#v, want tool payload", request)
	}
	if request.Headers["Authorization"] != "Bearer <redacted>" {
		t.Fatalf("Authorization preview = %q, want redacted bearer", request.Headers["Authorization"])
	}
	if request.Body.Model != "deepseek-v4-flash" || request.Body.MaxTokens != 64 || len(request.Body.Tools) != 1 || request.Body.ToolChoice == nil {
		t.Fatalf("preview body = %#v, want diagnostic tool body", request.Body)
	}
	if request.Body.Thinking["type"] != "disabled" {
		t.Fatalf("preview thinking = %#v, want disabled", request.Body.Thinking)
	}
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
	for _, want := range []string{"--model", "--catalog-model", "--smoke", "--tool-smoke", "--print-request", "--timeout", "--json", "Diagnose DeepSeek provider configuration"} {
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
