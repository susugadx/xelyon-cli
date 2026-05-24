package openrouter

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseOpenRouter_TextSmokeObservesUsageAndCost(t *testing.T) {
	var received openaicompat.ChatCompletionsRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-or-test" {
			t.Fatalf("Authorization = %q, want Bearer sk-or-test", r.Header.Get("Authorization"))
		}
		if r.Header.Get("HTTP-Referer") == "" || r.Header.Get("X-Title") != "XELYON CLI" {
			t.Fatalf("OpenRouter headers missing: referer=%q x-title=%q", r.Header.Get("HTTP-Referer"), r.Header.Get("X-Title"))
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeOpenRouterChatCompletionsSSE(w,
			`{"choices":[{"delta":{"content":"xelyon openrouter doctor ok"}}]}`,
			`{"choices":[{"delta":{}}],"usage":{"prompt_tokens":10,"completion_tokens":6,"prompt_tokens_details":{"cached_tokens":2},"completion_tokens_details":{"reasoning_tokens":1}}}`,
		)
	}))
	defer server.Close()

	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, server.URL)

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "openai/gpt-5.4",
		CatalogModel: "openai/gpt-5.4",
		RunSmoke:     true,
		TextSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	if received.Model != "openai/gpt-5.4" || received.MaxTokens != defaultOpenRouterDiagnosticSmokeMaxOutputTokens {
		t.Fatalf("request = %#v, want smoke model and max_tokens override", received)
	}
	if len(received.Tools) != 0 || received.ToolChoice != nil {
		t.Fatalf("text smoke request should not include tools: %#v", received)
	}
	if report.Smoke == nil || !report.Smoke.UsageObserved {
		t.Fatalf("Smoke = %#v, want observed usage", report.Smoke)
	}
	if report.Smoke.Usage.InputTokens != 10 || report.Smoke.Usage.OutputTokens != 5 || report.Smoke.Usage.CachedInputTokens != 2 || report.Smoke.Usage.ThinkingTokens != 1 {
		t.Fatalf("Smoke usage = %+v, want normalized usage", report.Smoke.Usage)
	}
	if report.Smoke.Cost.PricingUnavailable || report.Smoke.Cost.USD <= 0 {
		t.Fatalf("Smoke cost = %+v, want available positive estimate", report.Smoke.Cost)
	}
	for _, name := range []string{"smoke", "usage", "cost"} {
		requireOpenRouterDiagnosticCheckStatus(t, report, name, DiagnosticStatusOK)
	}
}

func TestDiagnoseOpenRouter_AnthropicSkinTextSmokeUsesMessagesPath(t *testing.T) {
	var requestPath string
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeOpenRouterAnthropicSSE(w,
			`{"type":"message_start","message":{"usage":{"input_tokens":10,"output_tokens":1,"cache_read_input_tokens":2,"cache_creation_input_tokens":1}}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"xelyon openrouter doctor ok"}}`,
			`{"type":"message_delta","usage":{"output_tokens":4}}`,
			`{"type":"message_stop"}`,
		)
	}))
	defer server.Close()

	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, server.URL+"/v1/chat/completions")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "anthropic/claude-sonnet-4.6",
		CatalogModel: "anthropic/claude-sonnet-4.6",
		RunSmoke:     true,
		TextSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	if requestPath != "/v1/messages" {
		t.Fatalf("request path = %q, want /v1/messages", requestPath)
	}
	if received["model"] != "anthropic/claude-sonnet-4.6" || received["context_management"] == nil {
		t.Fatalf("request body = %#v, want Claude context management body", received)
	}
	if report.Smoke == nil || !report.Smoke.UsageObserved {
		t.Fatalf("Smoke = %#v, want observed usage", report.Smoke)
	}
	if report.Smoke.Usage.InputTokens != 13 || report.Smoke.Usage.OutputTokens != 4 || report.Smoke.Usage.CachedInputTokens != 2 || report.Smoke.Usage.CacheCreationTokens != 1 {
		t.Fatalf("Smoke usage = %+v, want Anthropic normalized usage", report.Smoke.Usage)
	}
}

func TestDiagnoseOpenRouter_SmokeCostUsesRoutedModelWhenCatalogMismatches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeOpenRouterAnthropicSSE(w,
			`{"type":"message_start","message":{"usage":{"input_tokens":1000,"output_tokens":0}}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"xelyon openrouter doctor ok"}}`,
			`{"type":"message_delta","usage":{"output_tokens":0}}`,
			`{"type":"message_stop"}`,
		)
	}))
	defer server.Close()

	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, server.URL+"/v1/chat/completions")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "anthropic/claude-sonnet-4.6",
		CatalogModel: "openai/gpt-5.4",
		RunSmoke:     true,
		TextSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusWarn)
	requireOpenRouterDiagnosticCheckStatus(t, report, "cost", DiagnosticStatusOK)
	if report.Smoke == nil || report.Smoke.Cost.PricingUnavailable {
		t.Fatalf("Smoke = %#v, want priced smoke cost", report.Smoke)
	}
	if math.Abs(report.Smoke.Cost.USD-0.003) > 0.000000001 {
		t.Fatalf("Smoke cost = %.12f, want Claude routed-model price", report.Smoke.Cost.USD)
	}
}

func TestDiagnoseOpenRouter_SmokeEndpointFailureUsesCommonClassifier(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"route not found"}}`))
	}))
	defer server.Close()

	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, server.URL)

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "openai/gpt-5.4",
		CatalogModel: "openai/gpt-5.4",
		RunSmoke:     true,
		TextSmoke:    true,
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want endpoint-classified smoke failure: %#v", report.Checks)
	}
	smoke := requireOpenRouterDiagnosticCheckStatus(t, report, "smoke", DiagnosticStatusFail)
	if !strings.Contains(smoke.Message, "endpoint does not match") || !strings.Contains(smoke.Suggestion, openRouterAPIURLEnv) {
		t.Fatalf("smoke check = %#v, want common endpoint classification", smoke)
	}
}

func TestDiagnoseOpenRouter_SmokeEndpointOverrideResourceNotFoundUsesEndpointGuidance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"resource not found"}}`))
	}))
	defer server.Close()

	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, server.URL)

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "openai/gpt-5.4",
		CatalogModel: "openai/gpt-5.4",
		RunSmoke:     true,
		TextSmoke:    true,
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want endpoint-classified smoke failure: %#v", report.Checks)
	}
	smoke := requireOpenRouterDiagnosticCheckStatus(t, report, "smoke", DiagnosticStatusFail)
	if !strings.Contains(smoke.Message, "endpoint does not match") || !strings.Contains(smoke.Suggestion, openRouterAPIURLEnv) {
		t.Fatalf("smoke check = %#v, want endpoint override guidance", smoke)
	}
	if strings.Contains(smoke.Suggestion, "--model") {
		t.Fatalf("smoke suggestion = %q, should not suggest model changes for endpoint override resource errors", smoke.Suggestion)
	}
}

func TestDiagnoseOpenRouter_ToolSmokeRequiresToolCall(t *testing.T) {
	var received openaicompat.ChatCompletionsRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeOpenRouterChatCompletionsSSE(w,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"xelyon_openrouter_doctor_probe","arguments":"{\"value\":\"openrouter-tool-ok\"}"}}]}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":12,"completion_tokens":3}}`,
		)
	}))
	defer server.Close()

	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, server.URL)
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "openai/gpt-5.4",
		CatalogModel: "openai/gpt-5.4",
		RunSmoke:     true,
		ToolSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	if len(received.Tools) != 1 {
		t.Fatalf("tools = %#v, want diagnostic tool", received.Tools)
	}
	toolChoice, ok := received.ToolChoice.(map[string]any)
	if !ok {
		t.Fatalf("ToolChoice = %T, want forced function choice", received.ToolChoice)
	}
	function, ok := toolChoice["function"].(map[string]any)
	if !ok || function["name"] != openRouterDiagnosticSmokeToolName {
		t.Fatalf("ToolChoice = %#v, want diagnostic tool", received.ToolChoice)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "tool_smoke", DiagnosticStatusOK)
}

func TestDiagnoseOpenRouter_AnthropicSkinToolSmokeForcesToolChoice(t *testing.T) {
	var requestPath string
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeOpenRouterAnthropicSSE(w,
			`{"type":"message_start","message":{"usage":{"input_tokens":12,"output_tokens":1}}}`,
			`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"xelyon_openrouter_doctor_probe"}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"value\":\"openrouter-tool-ok\"}"}}`,
			`{"type":"content_block_stop","index":0}`,
			`{"type":"message_delta","usage":{"output_tokens":3}}`,
			`{"type":"message_stop"}`,
		)
	}))
	defer server.Close()

	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, server.URL+"/v1/chat/completions")
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "anthropic/claude-sonnet-4.6",
		CatalogModel: "anthropic/claude-sonnet-4.6",
		RunSmoke:     true,
		ToolSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	if requestPath != "/v1/messages" {
		t.Fatalf("request path = %q, want /v1/messages", requestPath)
	}
	toolChoice, ok := received["tool_choice"].(map[string]any)
	if !ok {
		t.Fatalf("tool_choice = %T, want forced Anthropic Skin tool choice", received["tool_choice"])
	}
	if toolChoice["type"] != "tool" || toolChoice["name"] != openRouterDiagnosticSmokeToolName {
		t.Fatalf("tool_choice = %#v, want diagnostic tool choice", toolChoice)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "tool_smoke", DiagnosticStatusOK)
}

func TestDiagnoseOpenRouter_ToolSmokeFailsWithoutToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeOpenRouterChatCompletionsSSE(w, `{"choices":[{"delta":{"content":"plain text"}}]}`)
	}))
	defer server.Close()

	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, server.URL)
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "openai/gpt-5.4",
		CatalogModel: "openai/gpt-5.4",
		RunSmoke:     true,
		ToolSmoke:    true,
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want tool smoke failure: %#v", report.Checks)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "tool_smoke", DiagnosticStatusFail)
}

func TestDiagnoseOpenRouter_FunctionCallingDisabledSkipsToolAndRunsTextFallback(t *testing.T) {
	requests := 0
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeOpenRouterChatCompletionsSSE(w,
			`{"choices":[{"delta":{"content":"fallback ok"}}]}`,
			`{"choices":[{"delta":{}}],"usage":{"prompt_tokens":8,"completion_tokens":2}}`,
		)
	}))
	defer server.Close()

	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, server.URL)
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "0")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "openai/gpt-5.4",
		CatalogModel: "openai/gpt-5.4",
		RunSmoke:     true,
		ToolSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want only text fallback network request", requests)
	}
	if _, ok := received["tools"]; ok {
		t.Fatalf("fallback request should not include tools: %#v", received["tools"])
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "tool_smoke", DiagnosticStatusWarn)
	if report.Smoke == nil || len(report.Smoke.Requests) != 2 || !report.Smoke.Requests[1].Skipped {
		t.Fatalf("Smoke requests = %#v, want text fallback plus skipped tool", report.Smoke)
	}
}
