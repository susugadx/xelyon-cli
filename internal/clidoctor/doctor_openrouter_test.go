package clidoctor

import (
	"bytes"
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
	requireDoctorContractTextContainsAll(t, out.String(), []string{
		"Upstream model: anthropic/claude-sonnet-4.6",
		"Route reason: request model anthropic/claude-sonnet-4.6 enables OpenRouter Anthropic Skin context management",
		"Request preview:",
		`"Authorization": "Bearer <redacted>"`,
		"Smoke route: anthropic_messages",
		"Smoke usage: input=10 cached=3 output=4 reasoning=2 cache_creation=1",
		"Smoke cost estimate: $0.00012345 USD",
	})
}
