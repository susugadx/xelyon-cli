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

	t.Setenv(geminiAPIKeyEnv, "gemini-key")
	t.Setenv(geminiAPIURLEnv, server.URL)
	t.Setenv("XELYON_MODEL", "")

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

	t.Setenv(geminiAPIKeyEnv, "gemini-key")
	t.Setenv(geminiAPIURLEnv, server.URL)
	t.Setenv("XELYON_MODEL", "")

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

	t.Setenv(geminiAPIKeyEnv, "gemini-key")
	t.Setenv(geminiAPIURLEnv, server.URL)
	t.Setenv("XELYON_MODEL", "")

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
	requireGeminiDiagnosticCheckStatus(t, report, "tool_smoke", DiagnosticStatusFail)
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

	t.Setenv(geminiAPIKeyEnv, "gemini-key")
	t.Setenv(geminiAPIURLEnv, server.URL)
	t.Setenv("XELYON_MODEL", "")

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
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"web search ok"}]},"groundingMetadata":{"groundingChunks":[{"web":{"uri":"https://example.com","title":"Example"}}]}}]}`))
	}))
	defer server.Close()

	t.Setenv(geminiAPIKeyEnv, "gemini-key")
	t.Setenv(geminiAPIURLEnv, server.URL)
	t.Setenv("XELYON_MODEL", "")

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
	if report.Smoke.UsageObserved {
		t.Fatalf("UsageObserved = true, want false for Gemini web search JSON path without usage callback")
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

func writeGeminiDiagnosticSSE(t *testing.T, w http.ResponseWriter, chunk GeminiFunctionResponse) {
	t.Helper()
	w.Header().Set("Content-Type", "text/event-stream")
	_, _ = w.Write([]byte(geminiSSEPayload(t, chunk)))
}
