package cmd

import (
	"bytes"
	"strings"
	"testing"

	kimiprovider "github.com/susugadx/xelyon-cli/internal/api/providers/kimi"
)

func TestRenderKimiDoctorText_WebSearchSmokeObservation(t *testing.T) {
	report := kimiprovider.DiagnosticReport{
		Provider:           "kimi",
		Model:              "kimi-k2.6",
		ModelSource:        "test",
		CatalogModel:       "kimi-k2.6",
		CatalogModelSource: "test",
		Route:              "chat_completions",
		APIURL:             "https://api.moonshot.ai/v1/chat/completions",
		Smoke: &kimiprovider.DiagnosticSmokeResult{
			Ran:                      true,
			WebSearchPayload:         true,
			Content:                  "ok",
			Duration:                 "10ms",
			UsageObserved:            true,
			CachedInputTokens:        3,
			WebSearchCallCount:       1,
			WebSearchCallFeeEstimate: 0.005,
			WebSearchUsageObserved:   true,
			SearchResultTotalTokens:  42,
		},
	}

	var out bytes.Buffer
	renderKimiDoctorText(&out, report)

	for _, want := range []string{
		"Web search call count: 1",
		"Web search call fee estimate: $0.0050 USD",
		"Web search usage observed: true",
		"Search result total tokens observed: 42",
		"call fee is separate from token cost",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("renderKimiDoctorText() output missing %q:\n%s", want, out.String())
		}
	}
}

func TestRenderKimiDoctorText_RequestPreview(t *testing.T) {
	report := kimiprovider.DiagnosticReport{
		Provider:           "kimi",
		Model:              "corp-kimi-model",
		ModelSource:        "test",
		CatalogModel:       "kimi-k2.6",
		CatalogModelSource: "test",
		Route:              "chat_completions",
		RouteReason:        "Kimi text, tool, image, and built-in $web_search diagnostics use Moonshot Chat Completions",
		APIURL:             "https://api.moonshot.ai/v1/chat/completions",
		RequestPreview: &kimiprovider.DiagnosticRequestPreview{
			Requests: []kimiprovider.DiagnosticRequestPreviewRequest{{
				Name:        "tool_smoke",
				ToolPayload: true,
				Route:       "chat_completions",
				Method:      "POST",
				URL:         "https://api.moonshot.ai/v1/chat/completions",
				Headers:     map[string]string{"Authorization": "Bearer <redacted>"},
				Body:        map[string]any{"model": "corp-kimi-model"},
			}},
		},
	}

	var out bytes.Buffer
	renderKimiDoctorText(&out, report)
	for _, want := range []string{
		"Catalog model: kimi-k2.6 (test)",
		"Route: chat_completions",
		"Route reason: Kimi text, tool, image, and built-in $web_search diagnostics use Moonshot Chat Completions",
		"Request preview:",
		`"Authorization": "Bearer <redacted>"`,
		`"tool_payload": true`,
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("renderKimiDoctorText() output missing %q:\n%s", want, out.String())
		}
	}
}

func TestRenderKimiDoctorText_SkippedToolSmokeRequest(t *testing.T) {
	report := kimiprovider.DiagnosticReport{
		Provider:    "kimi",
		Model:       "kimi-k2.6",
		ModelSource: "test",
		Route:       "chat_completions",
		APIURL:      "https://api.moonshot.ai/v1/chat/completions",
		Smoke: &kimiprovider.DiagnosticSmokeResult{
			Ran: true,
			Requests: []kimiprovider.DiagnosticSmokeRequestResult{{
				Name:        "tool_smoke",
				Skipped:     true,
				SkipReason:  "Kimi function calling payloads are disabled (KIMI_FUNCTION_CALLING=0)",
				ToolPayload: true,
			}},
		},
	}

	var out bytes.Buffer
	renderKimiDoctorText(&out, report)
	want := "Smoke request tool_smoke: skipped (Kimi function calling payloads are disabled (KIMI_FUNCTION_CALLING=0))"
	if !strings.Contains(out.String(), want) {
		t.Fatalf("renderKimiDoctorText() output missing %q:\n%s", want, out.String())
	}
}

func TestRunKimiDoctorInvocation_UsesConfiguredModelWhenFlagOmitted(t *testing.T) {
	setKimiDoctorCommandTestEnv(t, "moonshot-key")
	t.Setenv("XELYON_MODEL", "kimi-k2.5")

	cmd, out := newDoctorSubcommandTest(t, newKimiDoctorCommand)

	doctorJSONFlag = true

	if err := runKimiDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runKimiDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[struct {
		Model       string `json:"model"`
		ModelSource string `json:"model_source"`
	}](t, out)
	if report.Model != "kimi-k2.5" {
		t.Fatalf("model = %q, want XELYON_MODEL value kimi-k2.5", report.Model)
	}
	if report.ModelSource != "XELYON_MODEL" {
		t.Fatalf("model_source = %q, want XELYON_MODEL", report.ModelSource)
	}
}

func TestRunKimiDoctorInvocation_PrintRequestJSONReportsProxyEndpointWarning(t *testing.T) {
	proxyURL := "https://kimi.example/proxy"

	setKimiDoctorCommandTestEnv(t, "")
	t.Setenv("KIMI_API_URL", proxyURL)

	cmd, out := newDoctorSubcommandTest(t, newKimiDoctorCommand)

	if err := cmd.Flags().Set("model", "corp-kimi-model"); err != nil {
		t.Fatalf("set model flag: %v", err)
	}
	if err := cmd.Flags().Set("catalog-model", "kimi-k2.6"); err != nil {
		t.Fatalf("set catalog-model flag: %v", err)
	}
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runKimiDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runKimiDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	if report.APIURL != proxyURL {
		t.Fatalf("api_url = %q, want configured proxy URL", report.APIURL)
	}
	requireDoctorJSONProxyWarning(t, report.Checks, "api_url_path", "api_url", proxyURL)
	requireDoctorJSONRequestPreviewAllURLs(t, report.RequestPreview, proxyURL)
}

func TestRunKimiDoctorInvocation_FailsForMissingKey(t *testing.T) {
	setKimiDoctorCommandTestEnv(t, "")

	cmd, out := newDoctorSubcommandTest(t, newKimiDoctorCommand)

	err := runKimiDoctorInvocation(cmd, nil)
	if err == nil {
		t.Fatalf("runKimiDoctorInvocation() error = nil, want diagnostics failure\noutput:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "MOONSHOT_API_KEY") {
		t.Fatalf("output = %q, want MOONSHOT_API_KEY failure", out.String())
	}
}

func TestRootCommand_KimiDoctorCommandParsesFlags(t *testing.T) {
	setKimiDoctorCommandTestEnv(t, "moonshot-key")

	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "kimi", "--model", "corp-kimi-model", "--catalog-model", "kimi-k2.6", "--json"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"model": "corp-kimi-model"`) {
		t.Fatalf("output = %q, want parsed Kimi model", out.String())
	}
	if !strings.Contains(out.String(), `"catalog_model": "kimi-k2.6"`) {
		t.Fatalf("output = %q, want parsed Kimi catalog model", out.String())
	}
}

func TestRootCommand_KimiDoctorHelpShowsDoctorFlags(t *testing.T) {
	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "kimi", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "--model") {
		t.Fatalf("output = %q, want Kimi doctor model flag", out.String())
	}
	for _, want := range []string{"--catalog-model", "--tool-smoke", "--image-smoke", "--web-search-smoke", "--print-request", "exact Chat Completions", "endpoint override", "/v1/chat/completions"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output = %q, want Kimi doctor help substring %q", out.String(), want)
		}
	}
	if !strings.Contains(out.String(), "Diagnose Kimi native provider configuration") {
		t.Fatalf("output = %q, want Kimi doctor help", out.String())
	}
}

func setKimiDoctorCommandTestEnv(t *testing.T, apiKey string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MOONSHOT_API_KEY", apiKey)
	t.Setenv("KIMI_API_URL", "")
	t.Setenv("KIMI_FUNCTION_CALLING", "")
	t.Setenv("XELYON_MODEL", "")
}
