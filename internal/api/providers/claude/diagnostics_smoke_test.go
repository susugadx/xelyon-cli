package claude

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseClaude_TextSmokeObservesUsageAndCost(t *testing.T) {
	var captured Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		writeClaudeDiagnosticJSON(t, w, []Content{{Type: "text", Text: "xelyon claude doctor ok"}}, StreamUsage{
			InputTokens:              10,
			OutputTokens:             5,
			CacheReadInputTokens:     2,
			CacheCreationInputTokens: 1,
		})
	}))
	defer server.Close()

	setClaudeDiagnosticTestEnv(t, server.URL, "claude-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:          config.DefaultConfig(),
		Model:           defaultClaudeModel,
		CatalogModel:    defaultClaudeModel,
		RunSmoke:        true,
		TextSmoke:       true,
		MaxOutputTokens: 8,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if report.Smoke == nil || !report.Smoke.UsageObserved || report.Smoke.Usage.InputTokens != 13 || report.Smoke.Usage.OutputTokens != 5 || report.Smoke.Usage.CachedInputTokens != 2 || report.Smoke.Usage.CacheCreationTokens != 1 {
		t.Fatalf("Smoke = %#v, want observed normalized usage", report.Smoke)
	}
	if report.Smoke.Cost.PricingUnavailable || report.Smoke.Cost.USD <= 0 {
		t.Fatalf("Smoke cost = %+v, want available positive estimate", report.Smoke.Cost)
	}
	if len(captured.Tools) != 0 {
		t.Fatalf("tools = %#v, want absent for text smoke", captured.Tools)
	}
	if captured.MaxTokens != 8 || !captured.Stream {
		t.Fatalf("request max_tokens=%d stream=%t, want max_tokens 8 stream true", captured.MaxTokens, captured.Stream)
	}
}

func TestDiagnoseClaude_ToolSmokeRequiresToolCall(t *testing.T) {
	var captured Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		writeClaudeDiagnosticJSON(t, w, []Content{{
			Type:  "tool_use",
			ID:    "toolu_doctor",
			Name:  claudeDiagnosticToolName,
			Input: map[string]interface{}{"value": "claude-tool-ok"},
		}}, StreamUsage{InputTokens: 8, OutputTokens: 4})
	}))
	defer server.Close()

	setClaudeDiagnosticTestEnv(t, server.URL, "claude-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultClaudeModel,
		CatalogModel: defaultClaudeModel,
		RunSmoke:     true,
		ToolSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	requireClaudeDiagnosticCheckStatus(t, report, "tool_smoke", DiagnosticStatusOK)
	if report.Smoke == nil || !report.Smoke.ToolPayload || !strings.Contains(report.Smoke.Content, claudeDiagnosticToolName) {
		t.Fatalf("Smoke = %#v, want diagnostic tool call content", report.Smoke)
	}
	if len(captured.Tools) != 1 || captured.Tools[0].Name != claudeDiagnosticToolName {
		t.Fatalf("tools = %#v, want only diagnostic Claude tool", captured.Tools)
	}
	requireClaudeToolChoice(t, captured.ToolChoice, claudeDiagnosticToolName)
}

func TestDiagnoseClaude_ToolSmokeFailsWithoutToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeClaudeDiagnosticJSON(t, w, []Content{{Type: "text", Text: "plain response"}}, StreamUsage{})
	}))
	defer server.Close()

	setClaudeDiagnosticTestEnv(t, server.URL, "claude-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultClaudeModel,
		CatalogModel: defaultClaudeModel,
		RunSmoke:     true,
		ToolSmoke:    true,
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want tool smoke failure: %#v", report.Checks)
	}
	toolSmoke := requireClaudeDiagnosticCheckStatus(t, report, "tool_smoke", DiagnosticStatusFail)
	if !strings.Contains(toolSmoke.Detail, claudeDiagnosticToolName) {
		t.Fatalf("tool_smoke detail = %q, want diagnostic tool name", toolSmoke.Detail)
	}
	requireClaudeDiagnosticCheckStatus(t, report, "smoke", DiagnosticStatusFail)
}

func TestDiagnoseClaude_ToolSmokeDisabledRecordsSkippedRequestAndTextFallback(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		writeClaudeDiagnosticJSON(t, w, []Content{{Type: "text", Text: "fallback ok"}}, StreamUsage{InputTokens: 4, OutputTokens: 2})
	}))
	defer server.Close()

	setClaudeDiagnosticTestEnv(t, server.URL, "claude-key")
	t.Setenv(claudeFunctionCallEnv, "0")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultClaudeModel,
		CatalogModel: defaultClaudeModel,
		RunSmoke:     true,
		ToolSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false for skipped tool smoke: %#v", report.Checks)
	}
	requireClaudeDiagnosticCheckStatus(t, report, "tool_smoke", DiagnosticStatusWarn)
	if requestCount != 1 {
		t.Fatalf("requestCount = %d, want only text fallback request", requestCount)
	}
	if report.Smoke == nil || len(report.Smoke.Requests) != 2 {
		t.Fatalf("Smoke requests = %#v, want text fallback and skipped tool", report.Smoke)
	}
	if report.Smoke.Requests[0].Name != "text" || report.Smoke.Requests[1].Name != "tool" || !report.Smoke.Requests[1].Skipped {
		t.Fatalf("Smoke requests = %#v, want text then skipped tool", report.Smoke.Requests)
	}
	if report.Smoke.Requests[1].Ran {
		t.Fatalf("skipped tool request Ran = true, want false")
	}
}

func TestDiagnoseClaude_ImageSmokeBuildsImagePayload(t *testing.T) {
	var captured MultimodalRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		writeClaudeDiagnosticJSON(t, w, []Content{{Type: "text", Text: "image ok"}}, StreamUsage{InputTokens: 9, OutputTokens: 3})
	}))
	defer server.Close()

	setClaudeDiagnosticTestEnv(t, server.URL, "claude-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        defaultClaudeModel,
		CatalogModel: defaultClaudeModel,
		RunSmoke:     true,
		ImageSmoke:   true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	requireClaudeDiagnosticCheckStatus(t, report, "image_smoke", DiagnosticStatusOK)
	if report.Smoke == nil || !report.Smoke.ImagePayload || report.Smoke.Content != "image ok" {
		t.Fatalf("Smoke = %#v, want image smoke content", report.Smoke)
	}
	if len(captured.Tools) != 0 || len(captured.Messages) != 1 {
		t.Fatalf("request = %#v, want multimodal request without tools", captured)
	}
	user, ok := captured.Messages[0].(map[string]any)
	if !ok || user["role"] != "user" {
		t.Fatalf("user message = %#v, want decoded user map", captured.Messages[0])
	}
	content, ok := user["content"].([]any)
	if !ok || len(content) != 2 {
		t.Fatalf("content = %#v, want image and text parts", user["content"])
	}
	imagePart, ok := content[0].(map[string]any)
	if !ok || imagePart["type"] != "image" {
		t.Fatalf("image part = %#v, want image block", content[0])
	}
	source, ok := imagePart["source"].(map[string]any)
	if !ok || source["media_type"] != "image/png" || source["data"] != claudeDiagnosticPNGBase64 {
		t.Fatalf("source = %#v, want diagnostic PNG", imagePart["source"])
	}
}

func TestDiagnoseClaude_ThinkingSmokeEnablesThinkingConfig(t *testing.T) {
	var captured Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		writeClaudeDiagnosticJSON(t, w, []Content{{Type: "text", Text: "thinking ok"}}, StreamUsage{InputTokens: 7, OutputTokens: 3})
	}))
	defer server.Close()

	setClaudeDiagnosticTestEnv(t, server.URL, "claude-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:        config.DefaultConfig(),
		Model:         defaultClaudeModel,
		CatalogModel:  defaultClaudeModel,
		RunSmoke:      true,
		ThinkingSmoke: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	requireClaudeDiagnosticCheckStatus(t, report, "thinking_smoke", DiagnosticStatusOK)
	if captured.Thinking == nil {
		t.Fatalf("thinking = nil, want request-level thinking config")
	}
}

func TestDiagnoseClaude_WebSearchSmokeUsesNativeWebSearchPayload(t *testing.T) {
	const proxyPath = "/proxy"
	var captured webSearchRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != proxyPath {
			t.Fatalf("path = %s, want %s", r.URL.Path, proxyPath)
		}
		if r.Header.Get("anthropic-beta") != webSearchBetaHeader {
			t.Fatalf("anthropic-beta = %q, want %s", r.Header.Get("anthropic-beta"), webSearchBetaHeader)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"web search ok","citations":[{"title":"Example","url":"https://example.com"}]}]}`))
	}))
	defer server.Close()

	setClaudeDiagnosticTestEnv(t, server.URL+proxyPath, "claude-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:         config.DefaultConfig(),
		Model:          defaultClaudeModel,
		CatalogModel:   defaultClaudeModel,
		RunSmoke:       true,
		WebSearchSmoke: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	requireClaudeDiagnosticCheckStatus(t, report, "endpoint", DiagnosticStatusWarn)
	requireClaudeDiagnosticCheckStatus(t, report, "web_search_smoke", DiagnosticStatusOK)
	requireClaudeDiagnosticCheckStatus(t, report, "usage", DiagnosticStatusWarn)
	if report.Smoke == nil || report.Smoke.Route != DiagnosticRouteClaudeWebSearch || !report.Smoke.WebSearchPayload || !strings.Contains(report.Smoke.Content, "Summary:") || !strings.Contains(report.Smoke.Content, "Sources:") {
		t.Fatalf("Smoke = %#v, want web search summary and sources", report.Smoke)
	}
	if report.Smoke.UsageObserved {
		t.Fatalf("Smoke usage observed = true, want Claude web search smoke without token usage")
	}
	if len(captured.Tools) != 1 || captured.Tools[0].Type != "web_search_20250305" {
		t.Fatalf("tools = %#v, want one Claude web search tool", captured.Tools)
	}
}

func TestDiagnoseClaude_MultiSmokeRequiresUsageForEveryRanRequest(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		switch requestCount {
		case 1:
			writeClaudeDiagnosticJSON(t, w, []Content{{Type: "text", Text: "xelyon claude doctor ok"}}, StreamUsage{InputTokens: 10, OutputTokens: 5})
		case 2:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"web search ok","citations":[{"title":"Example","url":"https://example.com"}]}]}`))
		default:
			t.Fatalf("unexpected request %d", requestCount)
		}
	}))
	defer server.Close()

	setClaudeDiagnosticTestEnv(t, server.URL, "claude-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:         config.DefaultConfig(),
		Model:          defaultClaudeModel,
		CatalogModel:   defaultClaudeModel,
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
	requireClaudeDiagnosticCheckStatus(t, report, "usage", DiagnosticStatusWarn)
	requireClaudeDiagnosticCheckStatus(t, report, "cost", DiagnosticStatusWarn)
}

func writeClaudeDiagnosticJSON(t *testing.T, w http.ResponseWriter, content []Content, usage StreamUsage) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(Response{Content: content, Usage: usage}); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
