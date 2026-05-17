package cmd

import (
	"bytes"
	"strings"
	"testing"

	groqprovider "github.com/susugadx/xelyon-cli/internal/api/providers/groq"
)

func TestRunGroqDoctorInvocation_JSONReportsExplicitModelAndCatalogModel(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GROQ_API_KEY", "gsk-test")
	t.Setenv("GROQ_API_URL", "")

	cmd, out := newDoctorSubcommandTest(t, newGroqDoctorCommand)

	doctorGroqModelFlag = "corp-groq-model"
	doctorCatalogModelFlag = "meta-llama/llama-4-scout-17b-16e-instruct"
	doctorJSONFlag = true

	if err := runGroqDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runGroqDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	if report.Provider != "groq" {
		t.Fatalf("provider = %q, want groq", report.Provider)
	}
	if report.Model != "corp-groq-model" || report.ModelSource != "--model" {
		t.Fatalf("model = %q (%s), want explicit model", report.Model, report.ModelSource)
	}
	if report.CatalogModel != "meta-llama/llama-4-scout-17b-16e-instruct" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit catalog model", report.CatalogModel, report.CatalogModelSource)
	}
	if report.Route != "chat_completions" {
		t.Fatalf("route = %q, want chat_completions", report.Route)
	}
	catalogPolicy := requireDoctorJSONCheck(t, report.Checks, "catalog_policy")
	requireDoctorJSONCheckStatus(t, catalogPolicy, "ok")
	requireDoctorJSONCheckDetailContains(t, catalogPolicy, "max_output_tokens=8192")
}

func TestRunGroqDoctorInvocation_PrintRequestJSONDoesNotRequireAPIKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GROQ_API_KEY", "")
	t.Setenv("GROQ_API_URL", "")

	cmd, out := newDoctorSubcommandTest(t, newGroqDoctorCommand)

	doctorGroqModelFlag = "meta-llama/llama-4-scout-17b-16e-instruct"
	doctorCatalogModelFlag = "meta-llama/llama-4-scout-17b-16e-instruct"
	doctorToolSmokeFlag = true
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runGroqDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runGroqDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
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
		Model      string `json:"model"`
		MaxTokens  int    `json:"max_tokens"`
		Tools      []any  `json:"tools"`
		ToolChoice any    `json:"tool_choice"`
	}](t, request)
	if body.Model != "meta-llama/llama-4-scout-17b-16e-instruct" || body.MaxTokens != 64 || len(body.Tools) != 1 || body.ToolChoice == nil {
		t.Fatalf("preview body = %#v, want diagnostic tool body", body)
	}
}

func TestRunGroqDoctorInvocation_PrintRequestJSONReportsOpenAICompatibleProxyEndpointWarning(t *testing.T) {
	proxyURL := "https://groq.example/v1/chat/completions"
	model := "meta-llama/llama-4-scout-17b-16e-instruct"

	t.Setenv("HOME", t.TempDir())
	t.Setenv("GROQ_API_KEY", "")
	t.Setenv("GROQ_API_URL", proxyURL)

	cmd, out := newDoctorSubcommandTest(t, newGroqDoctorCommand)

	doctorGroqModelFlag = model
	doctorCatalogModelFlag = model
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runGroqDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runGroqDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	if report.APIURL != proxyURL {
		t.Fatalf("api_url = %q, want configured proxy URL", report.APIURL)
	}
	requireDoctorJSONProxyWarning(t, report.Checks, "endpoint", "", proxyURL)
	requireDoctorJSONRequestPreviewURLs(t, report.RequestPreview, 1, proxyURL)
}

func TestRootCommand_GroqDoctorCommandParsesFlags(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GROQ_API_KEY", "gsk-test")
	t.Setenv("GROQ_API_URL", "")

	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "groq", "--model", "corp-groq-model", "--catalog-model", "meta-llama/llama-4-scout-17b-16e-instruct", "--json"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"model": "corp-groq-model"`) {
		t.Fatalf("output = %q, want parsed Groq model", out.String())
	}
	if !strings.Contains(out.String(), `"catalog_model": "meta-llama/llama-4-scout-17b-16e-instruct"`) {
		t.Fatalf("output = %q, want parsed Groq catalog model", out.String())
	}
}

func TestRootCommand_GroqDoctorHelpShowsMinimalFlags(t *testing.T) {
	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "groq", "--help"})

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
		"Diagnose Groq provider configuration",
		"exact Chat Completions endpoint",
		"/openai/v1/chat/completions",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output = %q, want Groq doctor help substring %q", out.String(), want)
		}
	}
	for _, unwanted := range []string{"--capabilities", "--require-capability", "--retention-smoke", "--image-smoke", "--web-search-smoke", "--thinking-smoke", "--print-config"} {
		if strings.Contains(out.String(), unwanted) {
			t.Fatalf("output = %q, should not contain %s", out.String(), unwanted)
		}
	}
}

func TestRenderGroqDoctorTextIncludesRequestPreviewAndSmokeObservability(t *testing.T) {
	report := groqprovider.DiagnosticReport{
		Provider:           "groq",
		APIURL:             "https://api.groq.com/openai/v1/chat/completions",
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
		ModelSource:        "test",
		CatalogModel:       "meta-llama/llama-4-scout-17b-16e-instruct",
		CatalogModelSource: "test",
		Route:              groqprovider.DiagnosticRouteChatCompletions,
		RouteReason:        "Groq provider uses OpenAI-compatible Chat Completions",
		Checks: []groqprovider.DiagnosticCheck{
			{Name: "smoke", Status: groqprovider.DiagnosticStatusOK, Message: "live Groq smoke request succeeded"},
		},
		RequestPreview: &groqprovider.DiagnosticRequestPreview{
			Requests: []groqprovider.DiagnosticRequestPreviewRequest{{
				Name:    "text",
				Route:   groqprovider.DiagnosticRouteChatCompletions,
				Method:  "POST",
				URL:     "https://api.groq.com/openai/v1/chat/completions",
				Headers: map[string]string{"Authorization": "Bearer <redacted>"},
				Body:    map[string]any{"model": "meta-llama/llama-4-scout-17b-16e-instruct", "max_tokens": 64},
			}},
		},
		Smoke: &groqprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         groqprovider.DiagnosticRouteChatCompletions,
			Content:       "xelyon groq doctor ok",
			Duration:      "1ms",
			UsageObserved: true,
			Usage: groqprovider.DiagnosticSmokeUsage{
				InputTokens:         10,
				OutputTokens:        4,
				ThinkingTokens:      2,
				CachedInputTokens:   3,
				CacheCreationTokens: 1,
			},
			Cost: groqprovider.DiagnosticSmokeCost{
				USD: 0.00012345,
			},
		},
	}

	var out bytes.Buffer
	renderGroqDoctorText(&out, report)
	requireDoctorContractTextContainsAll(t, out.String(), []string{
		"Route reason: Groq provider uses OpenAI-compatible Chat Completions",
		"Request preview:",
		`"Authorization": "Bearer <redacted>"`,
		"Smoke route: chat_completions",
		"Smoke usage: input=10 cached=3 output=4 reasoning=2 cache_creation=1",
		"Smoke cost estimate: $0.00012345 USD",
	})
}
