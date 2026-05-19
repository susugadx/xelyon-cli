package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseGemini_TextSmokeObservesUsageAndCost(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		writeGeminiDiagnosticSSE(t, w, GeminiFunctionResponse{
			Candidates: []GeminiFunctionCandidate{{
				Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "xelyon gemini doctor ok"}}},
			}},
			UsageMetadata: &GeminiUsageMetadata{
				PromptTokenCount:        10,
				CandidatesTokenCount:    5,
				ThoughtsTokenCount:      1,
				CachedContentTokenCount: 2,
			},
		})
	}))
	defer server.Close()

	setGeminiDiagnosticSmokeTestEnv(t, server.URL, "gemini-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:          config.DefaultConfig(),
		Model:           defaultGeminiDiagnosticModel,
		CatalogModel:    defaultGeminiDiagnosticModel,
		RunSmoke:        true,
		TextSmoke:       true,
		MaxOutputTokens: 8,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if report.Smoke == nil || !report.Smoke.UsageObserved || report.Smoke.Usage.InputTokens != 10 || report.Smoke.Usage.OutputTokens != 5 || report.Smoke.Usage.ThinkingTokens != 1 || report.Smoke.Usage.CachedInputTokens != 2 {
		t.Fatalf("Smoke = %#v, want observed normalized usage", report.Smoke)
	}
	if report.Smoke.Cost.PricingUnavailable || report.Smoke.Cost.USD <= 0 {
		t.Fatalf("Smoke cost = %+v, want available positive estimate", report.Smoke.Cost)
	}
	if _, ok := captured["tools"]; ok {
		t.Fatalf("tools = %#v, want absent for text smoke", captured["tools"])
	}
	gen, ok := captured["generationConfig"].(map[string]any)
	if !ok || gen["maxOutputTokens"] != float64(8) {
		t.Fatalf("generationConfig = %#v, want maxOutputTokens 8", captured["generationConfig"])
	}
}

func TestDiagnoseGemini_ToolSmokeRequiresToolCallAndUsesAnyMode(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		writeGeminiDiagnosticSSE(t, w, GeminiFunctionResponse{
			Candidates: []GeminiFunctionCandidate{{
				Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{
					FunctionCall: &api.GeminiFunctionCall{
						Name: geminiDiagnosticToolName,
						Args: map[string]any{"value": "gemini-tool-ok"},
					},
				}}},
			}},
			UsageMetadata: &GeminiUsageMetadata{PromptTokenCount: 8, CandidatesTokenCount: 4},
		})
	}))
	defer server.Close()

	setGeminiDiagnosticSmokeTestEnv(t, server.URL, "gemini-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultGeminiDiagnosticModel,
		CatalogModel: defaultGeminiDiagnosticModel,
		RunSmoke:     true,
		ToolSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	requireGeminiDiagnosticCheckStatus(t, report, "tool_smoke", DiagnosticStatusOK)
	if report.Smoke == nil || !report.Smoke.ToolPayload || !strings.Contains(report.Smoke.Content, geminiDiagnosticToolName) {
		t.Fatalf("Smoke = %#v, want diagnostic tool call content", report.Smoke)
	}
	toolConfig, ok := captured["tool_config"].(map[string]any)
	if !ok {
		t.Fatalf("tool_config = %#v, want map", captured["tool_config"])
	}
	fcConfig, ok := toolConfig["function_calling_config"].(map[string]any)
	if !ok || fcConfig["mode"] != "ANY" {
		t.Fatalf("function_calling_config = %#v, want mode ANY", toolConfig["function_calling_config"])
	}
}

func TestDiagnoseGemini_ToolSmokeFailsWithoutToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeGeminiDiagnosticSSE(t, w, GeminiFunctionResponse{
			Candidates: []GeminiFunctionCandidate{{
				Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "plain response"}}},
			}},
		})
	}))
	defer server.Close()

	setGeminiDiagnosticSmokeTestEnv(t, server.URL, "gemini-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultGeminiDiagnosticModel,
		CatalogModel: defaultGeminiDiagnosticModel,
		RunSmoke:     true,
		ToolSmoke:    true,
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want tool smoke failure: %#v", report.Checks)
	}
	toolSmoke := requireGeminiDiagnosticCheckStatus(t, report, "tool_smoke", DiagnosticStatusFail)
	if !strings.Contains(toolSmoke.Suggestion, "function calling") {
		t.Fatalf("tool_smoke suggestion = %q, want function calling guidance", toolSmoke.Suggestion)
	}
	smoke := requireGeminiDiagnosticCheckStatus(t, report, "smoke", DiagnosticStatusFail)
	if !strings.Contains(smoke.Message, "tool smoke was not accepted") || !strings.Contains(smoke.Detail, "request=tool") {
		t.Fatalf("smoke check = %#v, want classified tool failure with request detail", smoke)
	}
}

func TestDiagnoseGemini_TextSmokeClassifiesAuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"API key not valid"}}`))
	}))
	defer server.Close()

	setGeminiDiagnosticSmokeTestEnv(t, server.URL, "bad-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultGeminiDiagnosticModel,
		CatalogModel: defaultGeminiDiagnosticModel,
		RunSmoke:     true,
		TextSmoke:    true,
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want auth-classified smoke failure: %#v", report.Checks)
	}
	smoke := requireGeminiDiagnosticCheckStatus(t, report, "smoke", DiagnosticStatusFail)
	if !strings.Contains(smoke.Message, "authentication or authorization") || !strings.Contains(smoke.Suggestion, geminiAPIKeyEnv) {
		t.Fatalf("smoke check = %#v, want auth failure guidance", smoke)
	}
	if report.Smoke == nil || len(report.Smoke.Requests) != 1 || !strings.Contains(report.Smoke.Requests[0].Error, "API key not valid") {
		t.Fatalf("Smoke = %#v, want request-level sanitized error", report.Smoke)
	}
}

func TestDiagnoseGemini_TextSmokeClassifiesCapacityFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"message":"Service Unavailable: backend overloaded"}}`))
	}))
	defer server.Close()

	setGeminiDiagnosticSmokeTestEnv(t, server.URL, "gemini-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultGeminiDiagnosticModel,
		CatalogModel: defaultGeminiDiagnosticModel,
		RunSmoke:     true,
		TextSmoke:    true,
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want capacity-classified smoke failure: %#v", report.Checks)
	}
	smoke := requireGeminiDiagnosticCheckStatus(t, report, "smoke", DiagnosticStatusFail)
	if !strings.Contains(smoke.Message, "quota, rate limit, or capacity") || !strings.Contains(smoke.Suggestion, "quota") {
		t.Fatalf("smoke check = %#v, want capacity guidance", smoke)
	}
}

func TestDiagnoseGemini_WebSearchSmokeClassifiesUnsupportedSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"google_search is unsupported for this model"}}`))
	}))
	defer server.Close()

	setGeminiDiagnosticSmokeTestEnv(t, server.URL, "gemini-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:         config.DefaultConfig(),
		Model:          defaultGeminiDiagnosticModel,
		CatalogModel:   defaultGeminiDiagnosticModel,
		RunSmoke:       true,
		WebSearchSmoke: true,
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want web search smoke failure: %#v", report.Checks)
	}
	webSearch := requireGeminiDiagnosticCheckStatus(t, report, "web_search_smoke", DiagnosticStatusFail)
	if !strings.Contains(webSearch.Suggestion, "google_search") {
		t.Fatalf("web_search_smoke suggestion = %q, want google_search guidance", webSearch.Suggestion)
	}
	smoke := requireGeminiDiagnosticCheckStatus(t, report, "smoke", DiagnosticStatusFail)
	if !strings.Contains(smoke.Message, "web search smoke was not accepted") || !strings.Contains(smoke.Detail, "request=web_search") {
		t.Fatalf("smoke check = %#v, want classified web search failure with request detail", smoke)
	}
}

func TestDiagnoseGemini_WebSearchSmokeClassifiesModelUnavailableBeforeWebSearchSupport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"models/missing is not found for API version v1beta, or is not supported for generateContent"}}`))
	}))
	defer server.Close()

	setGeminiDiagnosticSmokeTestEnv(t, server.URL, "gemini-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:         config.DefaultConfig(),
		Model:          "models/missing",
		CatalogModel:   defaultGeminiDiagnosticModel,
		RunSmoke:       true,
		WebSearchSmoke: true,
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want model-classified web search smoke failure: %#v", report.Checks)
	}
	smoke := requireGeminiDiagnosticCheckStatus(t, report, "smoke", DiagnosticStatusFail)
	if !strings.Contains(smoke.Message, "model is unavailable") || !strings.Contains(smoke.Suggestion, "--model") {
		t.Fatalf("smoke check = %#v, want model guidance before web search guidance", smoke)
	}
}

func TestDiagnoseGemini_ImageSmokeClassifiesImageUnsupportedBeforeEndpointHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"GenerateContentRequest.contents[0].parts[0].inline_data is unsupported"}}`))
	}))
	defer server.Close()

	setGeminiDiagnosticSmokeTestEnv(t, server.URL, "gemini-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultGeminiDiagnosticModel,
		CatalogModel: defaultGeminiDiagnosticModel,
		RunSmoke:     true,
		ImageSmoke:   true,
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want image-classified smoke failure: %#v", report.Checks)
	}
	smoke := requireGeminiDiagnosticCheckStatus(t, report, "smoke", DiagnosticStatusFail)
	if !strings.Contains(smoke.Message, "image smoke was not accepted") || !strings.Contains(smoke.Suggestion, "inline_data") {
		t.Fatalf("smoke check = %#v, want image guidance before endpoint guidance", smoke)
	}
	image := requireGeminiDiagnosticCheckStatus(t, report, "image_smoke", DiagnosticStatusFail)
	if !strings.Contains(image.Message, "image smoke failed before proving image input") {
		t.Fatalf("image_smoke check = %#v, want image-specific request failure", image)
	}
}

func TestDiagnoseGemini_ImageSmokeBuildsInlineDataPayload(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		writeGeminiDiagnosticSSE(t, w, GeminiFunctionResponse{
			Candidates: []GeminiFunctionCandidate{{
				Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "image ok"}}},
			}},
			UsageMetadata: &GeminiUsageMetadata{PromptTokenCount: 9, CandidatesTokenCount: 3},
		})
	}))
	defer server.Close()

	setGeminiDiagnosticSmokeTestEnv(t, server.URL, "gemini-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultGeminiDiagnosticModel,
		CatalogModel: defaultGeminiDiagnosticModel,
		RunSmoke:     true,
		ImageSmoke:   true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	requireGeminiDiagnosticCheckStatus(t, report, "image_smoke", DiagnosticStatusOK)
	if report.Smoke == nil || !report.Smoke.ImagePayload || report.Smoke.Content != "image ok" {
		t.Fatalf("Smoke = %#v, want image smoke content", report.Smoke)
	}
	if _, ok := captured["tools"]; ok {
		t.Fatalf("tools = %#v, want absent for image smoke", captured["tools"])
	}
	contents, ok := captured["contents"].([]any)
	if !ok || len(contents) != 1 {
		t.Fatalf("contents = %#v, want one multimodal user content", captured["contents"])
	}
	user, ok := contents[0].(map[string]any)
	if !ok {
		t.Fatalf("content = %#v, want map", contents[0])
	}
	parts, ok := user["parts"].([]any)
	if !ok || len(parts) != 2 {
		t.Fatalf("parts = %#v, want image and text parts", user["parts"])
	}
	imagePart, ok := parts[0].(map[string]any)
	if !ok {
		t.Fatalf("image part = %#v, want map", parts[0])
	}
	inline, ok := imagePart["inline_data"].(map[string]any)
	if !ok || inline["mime_type"] != "image/png" || inline["data"] != geminiDiagnosticPNGBase64 {
		t.Fatalf("inline_data = %#v, want diagnostic PNG", imagePart["inline_data"])
	}
}

func TestDiagnoseGemini_WebSearchSmokeUsesGenerateContentJSON(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"web search ok"}]},"groundingMetadata":{"groundingChunks":[{"web":{"uri":"https://example.com","title":"Example"}}]}}],"usageMetadata":{"promptTokenCount":12,"candidatesTokenCount":6,"thoughtsTokenCount":2,"cachedContentTokenCount":4}}`))
	}))
	defer server.Close()

	setGeminiDiagnosticSmokeTestEnv(t, server.URL, "gemini-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:         config.DefaultConfig(),
		Model:          defaultGeminiDiagnosticModel,
		CatalogModel:   defaultGeminiDiagnosticModel,
		RunSmoke:       true,
		WebSearchSmoke: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	requireGeminiDiagnosticCheckStatus(t, report, "web_search_smoke", DiagnosticStatusOK)
	if report.Smoke == nil || report.Smoke.Route != DiagnosticRouteGenerateContent || !report.Smoke.WebSearchPayload || !strings.Contains(report.Smoke.Content, "Summary:") || !strings.Contains(report.Smoke.Content, "Sources:") {
		t.Fatalf("Smoke = %#v, want web search summary and sources", report.Smoke)
	}
	if !report.Smoke.UsageObserved || report.Smoke.Usage.InputTokens != 12 || report.Smoke.Usage.OutputTokens != 6 || report.Smoke.Usage.ThinkingTokens != 2 || report.Smoke.Usage.CachedInputTokens != 4 {
		t.Fatalf("Smoke usage = %#v, want observed Gemini web search usageMetadata", report.Smoke)
	}
	if report.Smoke.Cost.PricingUnavailable || report.Smoke.Cost.USD <= 0 {
		t.Fatalf("Smoke cost = %+v, want available positive estimate", report.Smoke.Cost)
	}
	tools, ok := captured["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v, want one web search tool", captured["tools"])
	}
	tool, ok := tools[0].(map[string]any)
	if !ok || tool["google_search"] == nil {
		t.Fatalf("tool = %#v, want google_search", tools[0])
	}
}

func TestDiagnoseGemini_WebSearchSmokeSucceedsWithoutUsageMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"web search ok"}]}}]}`))
	}))
	defer server.Close()

	setGeminiDiagnosticSmokeTestEnv(t, server.URL, "gemini-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:         config.DefaultConfig(),
		Model:          defaultGeminiDiagnosticModel,
		CatalogModel:   defaultGeminiDiagnosticModel,
		RunSmoke:       true,
		WebSearchSmoke: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false without web search usageMetadata: %#v", report.Checks)
	}
	requireGeminiDiagnosticCheckStatus(t, report, "web_search_smoke", DiagnosticStatusOK)
	requireGeminiDiagnosticCheckStatus(t, report, "usage", DiagnosticStatusWarn)
	if report.Smoke == nil || report.Smoke.UsageObserved {
		t.Fatalf("Smoke = %#v, want web search success without usage observation", report.Smoke)
	}
}

func TestDiagnoseGemini_MultiSmokeRequiresUsageForEveryRanRequest(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		switch requestCount {
		case 1:
			writeGeminiDiagnosticSSE(t, w, GeminiFunctionResponse{
				Candidates: []GeminiFunctionCandidate{{
					Content: GeminiFunctionContent{Parts: []GeminiFunctionPart{{Text: "xelyon gemini doctor ok"}}},
				}},
				UsageMetadata: &GeminiUsageMetadata{PromptTokenCount: 10, CandidatesTokenCount: 5},
			})
		case 2:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"web search ok"}]}}]}`))
		default:
			t.Fatalf("unexpected request %d", requestCount)
		}
	}))
	defer server.Close()

	setGeminiDiagnosticSmokeTestEnv(t, server.URL, "gemini-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:         config.DefaultConfig(),
		Model:          defaultGeminiDiagnosticModel,
		CatalogModel:   defaultGeminiDiagnosticModel,
		RunSmoke:       true,
		TextSmoke:      true,
		WebSearchSmoke: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if report.Smoke == nil || len(report.Smoke.Requests) != 2 {
		t.Fatalf("Smoke = %#v, want text and web search requests", report.Smoke)
	}
	if !report.Smoke.Requests[0].UsageObserved || report.Smoke.Requests[1].UsageObserved {
		t.Fatalf("request usage observed = [%t,%t], want [true,false]", report.Smoke.Requests[0].UsageObserved, report.Smoke.Requests[1].UsageObserved)
	}
	if report.Smoke.UsageObserved {
		t.Fatalf("summary UsageObserved = true, want false when one ran request has no usage")
	}
	requireGeminiDiagnosticCheckStatus(t, report, "usage", DiagnosticStatusWarn)
	requireGeminiDiagnosticCheckStatus(t, report, "cost", DiagnosticStatusWarn)
}

func writeGeminiDiagnosticSSE(t *testing.T, w http.ResponseWriter, chunk GeminiFunctionResponse) {
	t.Helper()
	w.Header().Set("Content-Type", "text/event-stream")
	_, _ = w.Write([]byte(geminiSSEPayload(t, chunk)))
}

func setGeminiDiagnosticSmokeTestEnv(t *testing.T, apiURL, apiKey string) {
	t.Helper()
	t.Setenv(geminiAPIKeyEnv, apiKey)
	t.Setenv(geminiAPIURLEnv, apiURL)
	t.Setenv("XELYON_MODEL", "")
}
