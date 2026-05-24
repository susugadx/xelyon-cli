package cmd

import (
	"bytes"
	"strings"
	"testing"

	geminiprovider "github.com/susugadx/xelyon-cli/internal/api/providers/gemini"
)

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

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
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
	requireDoctorJSONPrintRequestOmittedSmoke(t, report.Smoke)
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

	requireDoctorJSONRequestPreviewCount(t, report.RequestPreview, 4)
	text := requireGeminiDoctorJSONPreviewRequest(t, report, 0, "text", "stream_generate_content_sse")
	textBody := requireDoctorJSONRequestPreviewBodyMap(t, text)
	requireDoctorJSONPreviewBodyAbsent(t, textBody, "tools", "tool_config")
	requireDoctorJSONPreviewBodyContains(t, textBody, "Reply with: xelyon gemini doctor ok")

	tool := requireGeminiDoctorJSONPreviewRequest(t, report, 1, "tool", "stream_generate_content_sse")
	if !tool.ToolPayload {
		t.Fatalf("tool request = %+v, want tool_payload", tool)
	}
	requireGeminiDoctorToolPreviewBody(t, requireDoctorJSONRequestPreviewBodyMap(t, tool))

	image := requireGeminiDoctorJSONPreviewRequest(t, report, 2, "image", "stream_generate_content_sse")
	if !image.ImagePayload {
		t.Fatalf("image request = %+v, want image_payload", image)
	}
	imageBody := requireDoctorJSONRequestPreviewBodyMap(t, image)
	requireGeminiDoctorImagePreviewBody(t, imageBody)
	requireDoctorJSONPreviewBodyAbsent(t, imageBody, "tools", "tool_config")

	web := requireGeminiDoctorJSONPreviewRequest(t, report, 3, "web_search", "generate_content")
	if !web.WebSearchPayload {
		t.Fatalf("web request = %+v, want web_search_payload", web)
	}
	webBody := requireDoctorJSONRequestPreviewBodyMap(t, web)
	requireGeminiDoctorWebSearchPreviewBody(t, webBody)
	requireDoctorJSONPreviewBodyAbsent(t, webBody, "tool_config")
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
				InputTokens:        22,
				CachedInputTokens:  6,
				OutputTokens:       10,
				ThinkingTokens:     3,
				BillingServiceTier: "standard",
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
						InputTokens:        10,
						CachedInputTokens:  2,
						OutputTokens:       4,
						ThinkingTokens:     1,
						BillingServiceTier: "standard",
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
						InputTokens:        12,
						CachedInputTokens:  4,
						OutputTokens:       6,
						ThinkingTokens:     2,
						BillingServiceTier: "standard",
					},
					Cost: geminiprovider.DiagnosticSmokeCost{USD: 0.00001},
				},
			},
		},
	}

	var out bytes.Buffer
	renderGeminiDoctorText(&out, report)
	requireDoctorContractTextContainsAll(t, out.String(), []string{
		"Gemini doctor",
		"Status: OK",
		"Capabilities: function_calling=true image_input=true web_search=true context_caching=true thinking=true",
		"Request preview:",
		`"web_search_payload": true`,
		"Smoke request text: ok route=stream_generate_content_sse duration=1ms",
		"Smoke usage text: input=10 cached=2 output=4 reasoning=1 cache_creation=0 billing_tier=standard",
		"Smoke cost estimate text: $0.00001000 USD",
		"Smoke request web_search: ok route=generate_content duration=3ms",
		"Smoke content web_search: Summary:\nweb search ok",
		"Smoke usage web_search: input=12 cached=4 output=6 reasoning=2 cache_creation=0 billing_tier=standard",
		"Smoke cost estimate web_search: $0.00001000 USD",
		"Smoke total usage: input=22 cached=6 output=10 reasoning=3 cache_creation=0 billing_tier=standard",
		"Smoke total cost estimate: $0.00002000 USD",
	})
}

func TestRenderGeminiDoctorTextContractWithSmokeFailure(t *testing.T) {
	report := geminiprovider.DiagnosticReport{
		Provider:               "gemini",
		APIURL:                 "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.1-pro-preview-customtools:streamGenerateContent?alt=sse",
		Model:                  "gemini-3.1-pro-preview-customtools",
		ModelSource:            "test",
		CatalogModel:           "gemini-3.1-pro-preview-customtools",
		CatalogModelSource:     "test",
		Route:                  geminiprovider.DiagnosticRouteStreamGenerateContentSSE,
		FunctionCallingEnabled: true,
		ImageInputSupported:    true,
		WebSearchSupported:     true,
		ContextCachingEnabled:  true,
		Checks: []geminiprovider.DiagnosticCheck{{
			Name:       "smoke",
			Status:     geminiprovider.DiagnosticStatusFail,
			Message:    "live Gemini smoke authentication or authorization failed",
			Detail:     "request=text route=stream_generate_content_sse error=API error (401): bad key",
			Suggestion: "Check GEMINI_API_KEY",
		}},
		Smoke: &geminiprovider.DiagnosticSmokeResult{
			Ran:      true,
			Route:    geminiprovider.DiagnosticRouteStreamGenerateContentSSE,
			Duration: "2ms",
			Requests: []geminiprovider.DiagnosticSmokeRequestResult{{
				Name:     "text",
				Ran:      true,
				Route:    geminiprovider.DiagnosticRouteStreamGenerateContentSSE,
				Duration: "2ms",
				Error:    "API error (401): bad key",
			}},
		},
	}

	var out bytes.Buffer
	renderGeminiDoctorText(&out, report)
	requireDoctorContractTextContainsAll(t, out.String(), []string{
		"Status: FAIL",
		"FAIL smoke: live Gemini smoke authentication or authorization failed",
		"detail: request=text route=stream_generate_content_sse error=API error (401): bad key",
		"suggestion: Check GEMINI_API_KEY",
		"Smoke error: API error (401): bad key",
	})
}

func requireGeminiDoctorJSONPreviewRequest(t *testing.T, report doctorJSONContractReport, index int, name, route string) doctorJSONRequestPreviewRequest {
	t.Helper()
	request := requireDoctorJSONRequestPreviewAt(t, report.RequestPreview, index, name)
	if request.Name != name || request.Route != route || request.Method != "POST" {
		t.Fatalf("request[%d] = %+v, want name=%s route=%s method=POST", index, request, name, route)
	}
	requireDoctorJSONRequestPreviewHeader(t, request, "Content-Type", "application/json")
	requireDoctorJSONRequestPreviewHeader(t, request, "x-goog-api-key", "<redacted>")
	return request
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
