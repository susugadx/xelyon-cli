package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	geminiprovider "github.com/susugadx/xelyon-cli/internal/api/providers/gemini"
)

type geminiDoctorJSONContractReport struct {
	Provider               string            `json:"provider"`
	APIURL                 string            `json:"api_url"`
	Model                  string            `json:"model"`
	ModelSource            string            `json:"model_source"`
	CatalogModel           string            `json:"catalog_model"`
	CatalogModelSource     string            `json:"catalog_model_source"`
	Route                  string            `json:"route"`
	RouteReason            string            `json:"route_reason"`
	MaxOutputTokens        int               `json:"max_output_tokens"`
	ContextWindowTokens    int               `json:"context_window_tokens"`
	FunctionCallingEnabled bool              `json:"function_calling_enabled"`
	ImageInputSupported    bool              `json:"image_input_supported"`
	WebSearchSupported     bool              `json:"web_search_supported"`
	ContextCachingEnabled  bool              `json:"context_caching_enabled"`
	ThinkingEnabled        bool              `json:"thinking_enabled"`
	Checks                 []doctorJSONCheck `json:"checks"`
	RequestPreview         struct {
		Requests []geminiDoctorJSONPreviewRequest `json:"requests"`
	} `json:"request_preview"`
	Smoke any `json:"smoke"`
}

type geminiDoctorJSONPreviewRequest struct {
	Name             string            `json:"name"`
	ToolPayload      bool              `json:"tool_payload"`
	ImagePayload     bool              `json:"image_payload"`
	WebSearchPayload bool              `json:"web_search_payload"`
	Route            string            `json:"route"`
	Method           string            `json:"method"`
	URL              string            `json:"url"`
	Headers          map[string]string `json:"headers"`
	Body             map[string]any    `json:"body"`
}

func TestRunGeminiDoctorInvocation_JSONContractPrintRequestAllShapes(t *testing.T) {
	setGeminiDoctorCommandTestEnv(t, "")

	cmd, out := newDoctorSubcommandTest(t, newGeminiDoctorCommand)
	doctorGeminiModelFlag = "gemini-3.1-pro-preview-customtools"
	doctorCatalogModelFlag = "gemini-3.1-pro-preview-customtools"
	doctorSmokeFlag = true
	doctorToolSmokeFlag = true
	doctorGeminiImageSmokeFlag = true
	doctorGeminiWebSearchSmokeFlag = true
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runGeminiDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runGeminiDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[geminiDoctorJSONContractReport](t, out)
	if report.Provider != "gemini" ||
		report.Model != "gemini-3.1-pro-preview-customtools" ||
		report.ModelSource != "--model" ||
		report.CatalogModel != "gemini-3.1-pro-preview-customtools" ||
		report.CatalogModelSource != "--catalog-model" ||
		report.Route != "stream_generate_content_sse" {
		t.Fatalf("Gemini doctor JSON identity fields = %+v", report)
	}
	if report.MaxOutputTokens != 65536 || report.ContextWindowTokens != 1000000 {
		t.Fatalf("Gemini doctor token policy = max %d context %d, want catalog values", report.MaxOutputTokens, report.ContextWindowTokens)
	}
	if !strings.Contains(report.APIURL, ":streamGenerateContent?alt=sse") ||
		!strings.Contains(report.RouteReason, "streamGenerateContent?alt=sse") ||
		!strings.Contains(report.RouteReason, "generateContent") {
		t.Fatalf("Gemini doctor route fields = api_url %q route_reason %q", report.APIURL, report.RouteReason)
	}
	if !report.FunctionCallingEnabled || !report.ImageInputSupported || !report.WebSearchSupported || !report.ContextCachingEnabled {
		t.Fatalf("Gemini doctor capability booleans = fc:%t image:%t web:%t cache:%t", report.FunctionCallingEnabled, report.ImageInputSupported, report.WebSearchSupported, report.ContextCachingEnabled)
	}
	if report.Smoke != nil {
		t.Fatalf("smoke = %#v, want omitted for --print-request", report.Smoke)
	}
	requireNoDoctorJSONChecks(t, report.Checks, "auth")
	for _, check := range []string{
		"endpoint",
		"provider_registration",
		"model",
		"catalog_model",
		"route",
		"catalog_policy",
		"function_calling",
		"image_input",
		"thinking",
		"context_caching",
		"web_search",
		"request_preview",
	} {
		requireDoctorJSONCheckStatus(t, requireDoctorJSONCheck(t, report.Checks, check), "ok")
	}

	if got, want := len(report.RequestPreview.Requests), 4; got != want {
		t.Fatalf("request_preview.requests length = %d, want %d", got, want)
	}
	text := requireGeminiDoctorJSONPreviewRequest(t, report, 0, "text", "stream_generate_content_sse")
	requireGeminiDoctorPreviewBodyAbsent(t, text.Body, "tools", "tool_config")
	requireGeminiDoctorPreviewBodyContainsText(t, text.Body, "Reply with: xelyon gemini doctor ok")

	tool := requireGeminiDoctorJSONPreviewRequest(t, report, 1, "tool", "stream_generate_content_sse")
	if !tool.ToolPayload {
		t.Fatalf("tool request = %+v, want tool_payload", tool)
	}
	requireGeminiDoctorToolPreviewBody(t, tool.Body)

	image := requireGeminiDoctorJSONPreviewRequest(t, report, 2, "image", "stream_generate_content_sse")
	if !image.ImagePayload {
		t.Fatalf("image request = %+v, want image_payload", image)
	}
	requireGeminiDoctorImagePreviewBody(t, image.Body)
	requireGeminiDoctorPreviewBodyAbsent(t, image.Body, "tools", "tool_config")

	web := requireGeminiDoctorJSONPreviewRequest(t, report, 3, "web_search", "generate_content")
	if !web.WebSearchPayload {
		t.Fatalf("web request = %+v, want web_search_payload", web)
	}
	requireGeminiDoctorWebSearchPreviewBody(t, web.Body)
	requireGeminiDoctorPreviewBodyAbsent(t, web.Body, "tool_config")
}

func TestRenderGeminiDoctorTextContractWithMultipleSmokeRequests(t *testing.T) {
	report := geminiprovider.DiagnosticReport{
		Provider:               "gemini",
		APIURL:                 "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.1-pro-preview-customtools:streamGenerateContent?alt=sse",
		Model:                  "gemini-3.1-pro-preview-customtools",
		ModelSource:            "test",
		CatalogModel:           "gemini-3.1-pro-preview-customtools",
		CatalogModelSource:     "test",
		Route:                  geminiprovider.DiagnosticRouteStreamGenerateContentSSE,
		RouteReason:            "Gemini text, tool, and image requests use streamGenerateContent?alt=sse; native web search uses generateContent",
		FunctionCallingEnabled: true,
		ImageInputSupported:    true,
		WebSearchSupported:     true,
		ContextCachingEnabled:  true,
		ThinkingEnabled:        true,
		Checks: []geminiprovider.DiagnosticCheck{
			{Name: "smoke", Status: geminiprovider.DiagnosticStatusOK, Message: "live Gemini smoke request succeeded"},
			{Name: "usage", Status: geminiprovider.DiagnosticStatusOK, Message: "Gemini smoke usage was observed", Detail: "input=22 cached_input=6 output=10 thinking=3 cache_creation=0"},
			{Name: "cost", Status: geminiprovider.DiagnosticStatusOK, Message: "Gemini smoke cost estimate is available", Detail: "$0.000020 USD"},
		},
		RequestPreview: &geminiprovider.DiagnosticRequestPreview{
			Requests: []geminiprovider.DiagnosticRequestPreviewRequest{{
				Name:             "web_search",
				WebSearchPayload: true,
				Route:            geminiprovider.DiagnosticRouteGenerateContent,
				Method:           "POST",
				URL:              "https://example.test/gemini",
				Headers:          map[string]string{"x-goog-api-key": "<redacted>"},
				Body:             map[string]any{"tools": []map[string]any{{"google_search": map[string]any{}}}},
			}},
		},
		Smoke: &geminiprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         "mixed",
			Content:       "xelyon gemini doctor ok",
			Duration:      "4ms",
			UsageObserved: true,
			Usage: geminiprovider.DiagnosticSmokeUsage{
				InputTokens:       22,
				CachedInputTokens: 6,
				OutputTokens:      10,
				ThinkingTokens:    3,
			},
			Cost: geminiprovider.DiagnosticSmokeCost{USD: 0.00002},
			Requests: []geminiprovider.DiagnosticSmokeRequestResult{
				{
					Name:          "text",
					Ran:           true,
					Route:         geminiprovider.DiagnosticRouteStreamGenerateContentSSE,
					Content:       "xelyon gemini doctor ok",
					Duration:      "1ms",
					UsageObserved: true,
					Usage: geminiprovider.DiagnosticSmokeUsage{
						InputTokens:       10,
						CachedInputTokens: 2,
						OutputTokens:      4,
						ThinkingTokens:    1,
					},
					Cost: geminiprovider.DiagnosticSmokeCost{USD: 0.00001},
				},
				{
					Name:             "web_search",
					Ran:              true,
					WebSearchPayload: true,
					Route:            geminiprovider.DiagnosticRouteGenerateContent,
					Content:          "Summary:\nweb search ok",
					Duration:         "3ms",
					UsageObserved:    true,
					Usage: geminiprovider.DiagnosticSmokeUsage{
						InputTokens:       12,
						CachedInputTokens: 4,
						OutputTokens:      6,
						ThinkingTokens:    2,
					},
					Cost: geminiprovider.DiagnosticSmokeCost{USD: 0.00001},
				},
			},
		},
	}

	var out bytes.Buffer
	renderGeminiDoctorText(&out, report)
	requireGeminiDoctorTextContainsAll(t, out.String(), []string{
		"Gemini doctor",
		"Status: OK",
		"Capabilities: function_calling=true image_input=true web_search=true context_caching=true thinking=true",
		"Request preview:",
		`"web_search_payload": true`,
		"Smoke request text: ok route=stream_generate_content_sse duration=1ms",
		"Smoke usage text: input=10 cached=2 output=4 reasoning=1 cache_creation=0",
		"Smoke cost estimate text: $0.00001000 USD",
		"Smoke request web_search: ok route=generate_content duration=3ms",
		"Smoke content web_search: Summary:\nweb search ok",
		"Smoke usage web_search: input=12 cached=4 output=6 reasoning=2 cache_creation=0",
		"Smoke cost estimate web_search: $0.00001000 USD",
		"Smoke total usage: input=22 cached=6 output=10 reasoning=3 cache_creation=0",
		"Smoke total cost estimate: $0.00002000 USD",
	})
}

func requireGeminiDoctorJSONPreviewRequest(t *testing.T, report geminiDoctorJSONContractReport, index int, name, route string) geminiDoctorJSONPreviewRequest {
	t.Helper()
	if index >= len(report.RequestPreview.Requests) {
		t.Fatalf("missing request index %d in %#v", index, report.RequestPreview.Requests)
	}
	request := report.RequestPreview.Requests[index]
	if request.Name != name || request.Route != route || request.Method != "POST" {
		t.Fatalf("request[%d] = %+v, want name=%s route=%s method=POST", index, request, name, route)
	}
	if request.Headers["Content-Type"] != "application/json" || request.Headers["x-goog-api-key"] != "<redacted>" {
		t.Fatalf("request[%d] headers = %#v, want redacted Gemini JSON headers", index, request.Headers)
	}
	return request
}

func requireGeminiDoctorPreviewBodyAbsent(t *testing.T, body map[string]any, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if _, ok := body[key]; ok {
			t.Fatalf("body should not contain %q: %#v", key, body)
		}
	}
}

func requireGeminiDoctorPreviewBodyContainsText(t *testing.T, body map[string]any, want string) {
	t.Helper()
	if !strings.Contains(renderedDoctorContractValue(t, body), want) {
		t.Fatalf("body = %#v, want text %q", body, want)
	}
}

func requireGeminiDoctorToolPreviewBody(t *testing.T, body map[string]any) {
	t.Helper()
	toolConfig, ok := body["tool_config"].(map[string]any)
	if !ok {
		t.Fatalf("tool_config = %#v, want object", body["tool_config"])
	}
	fcConfig, ok := toolConfig["function_calling_config"].(map[string]any)
	if !ok || fcConfig["mode"] != "ANY" {
		t.Fatalf("function_calling_config = %#v, want mode ANY", toolConfig["function_calling_config"])
	}
	if !strings.Contains(renderedDoctorContractValue(t, body["tools"]), "xelyon_gemini_doctor_probe") {
		t.Fatalf("tools = %#v, want diagnostic Gemini tool", body["tools"])
	}
}

func requireGeminiDoctorImagePreviewBody(t *testing.T, body map[string]any) {
	t.Helper()
	rendered := renderedDoctorContractValue(t, body)
	if !strings.Contains(rendered, "inline_data") || !strings.Contains(rendered, "image/png") {
		t.Fatalf("image body = %#v, want inline_data image/png", body)
	}
}

func requireGeminiDoctorWebSearchPreviewBody(t *testing.T, body map[string]any) {
	t.Helper()
	rendered := renderedDoctorContractValue(t, body)
	if !strings.Contains(rendered, "google_search") || strings.Contains(rendered, "google_search_retrieval") {
		t.Fatalf("web search body = %#v, want google_search request", body)
	}
}

func requireGeminiDoctorTextContainsAll(t *testing.T, output string, wants []string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func renderedDoctorContractValue(t *testing.T, value any) string {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%T) error = %v", value, err)
	}
	return string(payload)
}
