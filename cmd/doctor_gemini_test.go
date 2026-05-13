package cmd

import (
	"bytes"
	"strings"
	"testing"

	geminiprovider "github.com/susugadx/xelyon-cli/internal/api/providers/gemini"
)

func TestRunGeminiDoctorInvocation_JSONReportsExplicitModelCatalogAndRoutes(t *testing.T) {
	setGeminiDoctorCommandTestEnv(t, "gemini-key")

	cmd, out := newDoctorSubcommandTest(t, newGeminiDoctorCommand)

	doctorGeminiModelFlag = "corp-gemini-model"
	doctorCatalogModelFlag = "gemini-3.1-pro-preview-customtools"
	doctorJSONFlag = true

	if err := runGeminiDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runGeminiDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[struct {
		Provider               string            `json:"provider"`
		Model                  string            `json:"model"`
		ModelSource            string            `json:"model_source"`
		CatalogModel           string            `json:"catalog_model"`
		CatalogModelSource     string            `json:"catalog_model_source"`
		Route                  string            `json:"route"`
		FunctionCallingEnabled bool              `json:"function_calling_enabled"`
		ImageInputSupported    bool              `json:"image_input_supported"`
		WebSearchSupported     bool              `json:"web_search_supported"`
		Checks                 []doctorJSONCheck `json:"checks"`
	}](t, out)
	if report.Provider != "gemini" {
		t.Fatalf("provider = %q, want gemini", report.Provider)
	}
	if report.Model != "corp-gemini-model" || report.ModelSource != "--model" {
		t.Fatalf("model = %q (%s), want explicit model", report.Model, report.ModelSource)
	}
	if report.CatalogModel != "gemini-3.1-pro-preview-customtools" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit catalog model", report.CatalogModel, report.CatalogModelSource)
	}
	if report.Route != "stream_generate_content_sse" {
		t.Fatalf("route = %q, want stream_generate_content_sse", report.Route)
	}
	if !report.FunctionCallingEnabled || !report.ImageInputSupported || !report.WebSearchSupported {
		t.Fatalf("capabilities = fc:%t image:%t web:%t, want all enabled", report.FunctionCallingEnabled, report.ImageInputSupported, report.WebSearchSupported)
	}
	catalogPolicy := requireDoctorJSONCheck(t, report.Checks, "catalog_policy")
	requireDoctorJSONCheckStatus(t, catalogPolicy, "ok")
	requireDoctorJSONCheckDetailContains(t, catalogPolicy, "max_output_tokens=65536")
}

func TestRunGeminiDoctorInvocation_PrintRequestJSONDoesNotRequireAPIKey(t *testing.T) {
	setGeminiDoctorCommandTestEnv(t, "")

	cmd, out := newDoctorSubcommandTest(t, newGeminiDoctorCommand)

	doctorGeminiModelFlag = "gemini-3.1-pro-preview-customtools"
	doctorCatalogModelFlag = "gemini-3.1-pro-preview-customtools"
	doctorToolSmokeFlag = true
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runGeminiDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runGeminiDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[struct {
		Smoke          any `json:"smoke"`
		RequestPreview struct {
			Requests []struct {
				Name        string            `json:"name"`
				ToolPayload bool              `json:"tool_payload"`
				Route       string            `json:"route"`
				URL         string            `json:"url"`
				Headers     map[string]string `json:"headers"`
				Body        struct {
					ToolConfig struct {
						FunctionCallingConfig struct {
							Mode string `json:"mode"`
						} `json:"function_calling_config"`
					} `json:"tool_config"`
					Tools []struct {
						FunctionDeclarations []struct {
							Name string `json:"name"`
						} `json:"function_declarations"`
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
	if request.Name != "tool" || !request.ToolPayload || request.Route != "stream_generate_content_sse" {
		t.Fatalf("preview request = %#v, want Gemini tool stream request", request)
	}
	if request.Headers["x-goog-api-key"] != "<redacted>" {
		t.Fatalf("x-goog-api-key preview = %q, want redacted", request.Headers["x-goog-api-key"])
	}
	if request.Body.ToolConfig.FunctionCallingConfig.Mode != "ANY" {
		t.Fatalf("tool mode = %q, want ANY", request.Body.ToolConfig.FunctionCallingConfig.Mode)
	}
	if len(request.Body.Tools) != 1 || len(request.Body.Tools[0].FunctionDeclarations) != 1 || request.Body.Tools[0].FunctionDeclarations[0].Name != "xelyon_gemini_doctor_probe" {
		t.Fatalf("tools = %#v, want diagnostic Gemini tool", request.Body.Tools)
	}
}

func TestRootCommand_GeminiDoctorCommandParsesFlags(t *testing.T) {
	setGeminiDoctorCommandTestEnv(t, "gemini-key")

	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "gemini", "--model", "corp-gemini-model", "--catalog-model", "gemini-3.1-pro-preview-customtools", "--json"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"model": "corp-gemini-model"`) {
		t.Fatalf("output = %q, want parsed Gemini model", out.String())
	}
	if !strings.Contains(out.String(), `"catalog_model": "gemini-3.1-pro-preview-customtools"`) {
		t.Fatalf("output = %q, want parsed Gemini catalog model", out.String())
	}
}

func setGeminiDoctorCommandTestEnv(t *testing.T, apiKey string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GEMINI_API_KEY", apiKey)
	t.Setenv("GEMINI_API_URL", "")
	t.Setenv("XELYON_MODEL", "")
}

func TestRootCommand_GeminiDoctorHelpShowsMinimalFlags(t *testing.T) {
	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "gemini", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	for _, want := range []string{"--model", "--catalog-model", "--smoke", "--tool-smoke", "--image-smoke", "--web-search-smoke", "--print-request", "--timeout", "--json", "Diagnose Gemini provider configuration"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output = %q, want Gemini doctor help substring %q", out.String(), want)
		}
	}
	for _, unwanted := range []string{"--capabilities", "--require-capability", "--retention-smoke", "--thinking-smoke", "--print-config"} {
		if strings.Contains(out.String(), unwanted) {
			t.Fatalf("output = %q, should not contain %s", out.String(), unwanted)
		}
	}
}

func TestRenderGeminiDoctorTextIncludesRequestPreviewAndSmokeObservability(t *testing.T) {
	report := geminiprovider.DiagnosticReport{
		Provider:               "gemini",
		APIURL:                 "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.1-pro-preview-customtools:streamGenerateContent?alt=sse",
		Model:                  "gemini-3.1-pro-preview-customtools",
		ModelSource:            "test",
		CatalogModel:           "gemini-3.1-pro-preview-customtools",
		CatalogModelSource:     "test",
		Route:                  geminiprovider.DiagnosticRouteStreamGenerateContentSSE,
		RouteReason:            "Gemini text, tool, and image requests use streamGenerateContent?alt=sse",
		FunctionCallingEnabled: true,
		ImageInputSupported:    true,
		WebSearchSupported:     true,
		ContextCachingEnabled:  true,
		ThinkingEnabled:        false,
		Checks: []geminiprovider.DiagnosticCheck{
			{Name: "smoke", Status: geminiprovider.DiagnosticStatusOK, Message: "live Gemini smoke request succeeded"},
		},
		RequestPreview: &geminiprovider.DiagnosticRequestPreview{
			Requests: []geminiprovider.DiagnosticRequestPreviewRequest{{
				Name:    "text",
				Route:   geminiprovider.DiagnosticRouteStreamGenerateContentSSE,
				Method:  "POST",
				URL:     "https://example.test/gemini",
				Headers: map[string]string{"x-goog-api-key": "<redacted>"},
				Body:    map[string]any{"model": "gemini-3.1-pro-preview-customtools"},
			}},
		},
		Smoke: &geminiprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         geminiprovider.DiagnosticRouteStreamGenerateContentSSE,
			Content:       "xelyon gemini doctor ok",
			Duration:      "1ms",
			UsageObserved: true,
			Usage: geminiprovider.DiagnosticSmokeUsage{
				InputTokens:         10,
				OutputTokens:        4,
				ThinkingTokens:      1,
				CachedInputTokens:   3,
				CacheCreationTokens: 0,
			},
			Cost: geminiprovider.DiagnosticSmokeCost{
				USD: 0.00012345,
			},
		},
	}

	var out bytes.Buffer
	renderGeminiDoctorText(&out, report)
	output := out.String()
	for _, want := range []string{
		"Gemini doctor",
		"Route: stream_generate_content_sse",
		"Capabilities: function_calling=true image_input=true web_search=true context_caching=true thinking=false",
		"Request preview:",
		`"x-goog-api-key": "<redacted>"`,
		"Smoke route: stream_generate_content_sse",
		"Smoke usage: input=10 cached=3 output=4 reasoning=1 cache_creation=0",
		"Smoke cost estimate: $0.00012345 USD",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}
}
