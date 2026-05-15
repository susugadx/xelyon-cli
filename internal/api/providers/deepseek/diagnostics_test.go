package deepseek

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseDeepSeek_MissingAPIKeyFails(t *testing.T) {
	t.Setenv(deepSeekAPIKeyEnv, "")
	t.Setenv(deepSeekAPIURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want missing API key failure: %#v", report.Checks)
	}
	check, ok := deepSeekDiagnosticCheckByName(report, "auth")
	if !ok || check.Status != DiagnosticStatusFail {
		t.Fatalf("auth check = %#v, %v; want fail", check, ok)
	}
}

func TestDiagnoseDeepSeek_ModelCatalogPolicyRouteThinkingAndFunctionCalling(t *testing.T) {
	t.Setenv(deepSeekAPIKeyEnv, "sk-test")
	t.Setenv(deepSeekAPIURLEnv, "")
	t.Setenv(deepSeekFunctionCallingEnv, "1")

	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "xhigh"

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       cfg,
		Model:        "corp-deepseek-model",
		CatalogModel: "deepseek-v4-flash",
	})

	if report.Provider != "deepseek" {
		t.Fatalf("Provider = %q, want deepseek", report.Provider)
	}
	if report.Model != "corp-deepseek-model" || report.ModelSource != "--model" {
		t.Fatalf("model = %q (%s), want explicit model", report.Model, report.ModelSource)
	}
	if report.APIModel != "corp-deepseek-model" {
		t.Fatalf("APIModel = %q, want request alias preserved", report.APIModel)
	}
	if report.CatalogModel != "deepseek-v4-flash" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit catalog model", report.CatalogModel, report.CatalogModelSource)
	}
	if report.Route != DiagnosticRouteChatCompletions {
		t.Fatalf("Route = %q, want chat_completions", report.Route)
	}
	if report.MaxOutputTokens != 384000 {
		t.Fatalf("MaxOutputTokens = %d, want DeepSeek catalog max output", report.MaxOutputTokens)
	}
	if report.ContextWindowTokens != 1000000 {
		t.Fatalf("ContextWindowTokens = %d, want DeepSeek V4 context window", report.ContextWindowTokens)
	}
	if !report.FunctionCallingEnabled {
		t.Fatal("FunctionCallingEnabled = false, want true")
	}
	if !report.ThinkingSupported || !report.ThinkingEnabled || report.ThinkingType != "enabled" || report.ReasoningEffort != "max" {
		t.Fatalf("thinking = supported:%t enabled:%t type:%q effort:%q, want V4 xhigh enabled", report.ThinkingSupported, report.ThinkingEnabled, report.ThinkingType, report.ReasoningEffort)
	}
	for _, name := range []string{"auth", "endpoint", "provider_registration", "model", "catalog_model", "route", "catalog_policy", "thinking", "function_calling"} {
		check, ok := deepSeekDiagnosticCheckByName(report, name)
		if !ok || check.Status != DiagnosticStatusOK {
			t.Fatalf("%s check = %#v, %v; want ok", name, check, ok)
		}
	}
	catalogPolicy, _ := deepSeekDiagnosticCheckByName(report, "catalog_policy")
	if !strings.Contains(catalogPolicy.Detail, "context_window=1000000") ||
		!strings.Contains(catalogPolicy.Detail, "max_output_tokens=384000") ||
		!strings.Contains(catalogPolicy.Detail, "pricing=input $0.14/M") {
		t.Fatalf("catalog_policy detail = %q, want DeepSeek policy detail", catalogPolicy.Detail)
	}
}

func TestDiagnoseDeepSeek_NonDeepSeekCatalogModelDoesNotUseGlobalMetadata(t *testing.T) {
	t.Setenv(deepSeekAPIKeyEnv, "sk-test")
	t.Setenv(deepSeekAPIURLEnv, "")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("deepseek", config.ProviderModelConfig{
		DefaultModel: "corp-deepseek-model",
		CatalogModel: "deepseek-v4-flash",
	})

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       cfg,
		Model:        "corp-deepseek-model",
		CatalogModel: "gpt-5.5",
	})

	if report.CatalogModel != "gpt-5.5" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit value in report", report.CatalogModel, report.CatalogModelSource)
	}
	if report.ContextWindowTokens != 0 {
		t.Fatalf("ContextWindowTokens = %d, want non-DeepSeek catalog metadata ignored", report.ContextWindowTokens)
	}
	if report.MaxOutputTokens == 128000 {
		t.Fatalf("MaxOutputTokens = %d, want OpenAI catalog max output ignored", report.MaxOutputTokens)
	}
	catalogModel, ok := deepSeekDiagnosticCheckByName(report, "catalog_model")
	if !ok || catalogModel.Status != DiagnosticStatusWarn {
		t.Fatalf("catalog_model check = %#v, %v; want warn", catalogModel, ok)
	}
	catalogPolicy, ok := deepSeekDiagnosticCheckByName(report, "catalog_policy")
	if !ok || catalogPolicy.Status != DiagnosticStatusWarn {
		t.Fatalf("catalog_policy check = %#v, %v; want warn", catalogPolicy, ok)
	}
	if !strings.Contains(catalogPolicy.Detail, "context_window=unknown") ||
		!strings.Contains(catalogPolicy.Detail, "max_output_tokens=unknown") ||
		strings.Contains(catalogPolicy.Detail, "1050000") ||
		strings.Contains(catalogPolicy.Detail, "128000") {
		t.Fatalf("catalog_policy detail = %q, want no OpenAI token metadata", catalogPolicy.Detail)
	}
}

func TestDiagnoseDeepSeek_PrintRequestDoesNotRequireAPIKeyOrSendNetwork(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Fatalf("unexpected network request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	t.Setenv(deepSeekAPIKeyEnv, "")
	t.Setenv(deepSeekAPIURLEnv, server.URL+"/chat/completions")
	t.Setenv(deepSeekFunctionCallingEnv, "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "deepseek-chat",
		CatalogModel: "deepseek-chat",
		ToolSmoke:    true,
		PrintRequest: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want print request without API key to pass: %#v", report.Checks)
	}
	if _, ok := deepSeekDiagnosticCheckByName(report, "auth"); ok {
		t.Fatalf("auth check was added for print-request report: %#v", report.Checks)
	}
	if report.Smoke != nil {
		t.Fatalf("Smoke = %#v, want nil for print-request", report.Smoke)
	}
	if requests != 0 {
		t.Fatalf("network requests = %d, want 0", requests)
	}
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 1 {
		t.Fatalf("RequestPreview = %#v, want one tool request", report.RequestPreview)
	}
	preview := report.RequestPreview.Requests[0]
	if preview.Name != "tool" || !preview.ToolPayload || preview.URL != server.URL+"/chat/completions" {
		t.Fatalf("preview = %#v, want tool request to DeepSeek endpoint", preview)
	}
	if preview.Headers["Authorization"] != "Bearer <redacted>" {
		t.Fatalf("Authorization preview = %q, want redacted bearer", preview.Headers["Authorization"])
	}
	body := decodeDeepSeekDiagnosticPreviewBody(t, preview.Body)
	if body["model"] != "deepseek-v4-flash" {
		t.Fatalf("preview model = %q, want normalized deepseek-v4-flash", body["model"])
	}
	if tools, ok := body["tools"].([]any); !ok || len(tools) != 1 || body["tool_choice"] == nil {
		t.Fatalf("preview body = %#v, want forced diagnostic tool payload", body)
	}
	thinking, ok := body["thinking"].(map[string]any)
	if !ok || thinking["type"] != "disabled" {
		t.Fatalf("preview thinking = %#v, want disabled V4 thinking payload", body["thinking"])
	}
}

func TestDiagnoseDeepSeek_PrintRequestSkipsDisabledToolPreview(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Fatalf("unexpected network request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	t.Setenv(deepSeekAPIKeyEnv, "")
	t.Setenv(deepSeekAPIURLEnv, server.URL+"/chat/completions")
	t.Setenv(deepSeekFunctionCallingEnv, "0")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "deepseek-v4-flash",
		CatalogModel: "deepseek-v4-flash",
		ToolSmoke:    true,
		PrintRequest: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want disabled tool preview skip to pass: %#v", report.Checks)
	}
	if requests != 0 {
		t.Fatalf("network requests = %d, want 0", requests)
	}
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 2 {
		t.Fatalf("RequestPreview = %#v, want text fallback plus skipped tool request", report.RequestPreview)
	}
	text := report.RequestPreview.Requests[0]
	if text.Name != "text" || text.Skipped || text.ToolPayload {
		t.Fatalf("text preview = %#v, want runnable text fallback", text)
	}
	tool := report.RequestPreview.Requests[1]
	if tool.Name != "tool" || !tool.Skipped || !tool.ToolPayload || !strings.Contains(tool.SkipReason, deepSeekFunctionCallingEnv+"=0") {
		t.Fatalf("tool preview = %#v, want skipped disabled tool request", tool)
	}
}

func TestDiagnoseDeepSeek_TextSmokeObservesUsageAndCost(t *testing.T) {
	var received openaicompat.ChatCompletionsRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Fatalf("Authorization = %q, want Bearer sk-test", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeDeepSeekSSE(w,
			`{"choices":[{"delta":{"content":"xelyon deepseek doctor ok"}}]}`,
			`{"choices":[],"usage":{"prompt_tokens":10,"completion_tokens":6,"prompt_cache_hit_tokens":2,"completion_tokens_details":{"reasoning_tokens":2}}}`,
		)
	}))
	defer server.Close()

	t.Setenv(deepSeekAPIKeyEnv, "sk-test")
	t.Setenv(deepSeekAPIURLEnv, server.URL+"/chat/completions")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "deepseek-v4-flash",
		CatalogModel: "deepseek-v4-flash",
		RunSmoke:     true,
		TextSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	if received.Model != "deepseek-v4-flash" || received.MaxTokens != defaultDeepSeekDiagnosticSmokeMaxOutputTokens {
		t.Fatalf("request = %#v, want smoke model and max_tokens override", received)
	}
	if len(received.Tools) != 0 || received.ToolChoice != nil {
		t.Fatalf("text smoke request should not include tools: %#v", received)
	}
	if report.Smoke == nil || !report.Smoke.UsageObserved {
		t.Fatalf("Smoke = %#v, want observed usage", report.Smoke)
	}
	if report.Smoke.Usage.InputTokens != 10 ||
		report.Smoke.Usage.OutputTokens != 4 ||
		report.Smoke.Usage.ThinkingTokens != 2 ||
		report.Smoke.Usage.CachedInputTokens != 2 {
		t.Fatalf("Smoke usage = %+v, want normalized DeepSeek usage", report.Smoke.Usage)
	}
	if report.Smoke.Cost.PricingUnavailable || report.Smoke.Cost.USD <= 0 {
		t.Fatalf("Smoke cost = %+v, want available positive estimate", report.Smoke.Cost)
	}
	for _, name := range []string{"smoke", "usage", "cost"} {
		check, ok := deepSeekDiagnosticCheckByName(report, name)
		if !ok || check.Status != DiagnosticStatusOK {
			t.Fatalf("%s check = %#v, %v; want ok", name, check, ok)
		}
	}
}

func TestDiagnoseDeepSeek_ToolSmokeRequiresToolCall(t *testing.T) {
	var received openaicompat.ChatCompletionsRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeDeepSeekSSE(w,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"xelyon_deepseek_doctor_probe","arguments":"{\"value\":\"deepseek-tool-ok\"}"}}]}}]}`,
			`{"choices":[],"usage":{"prompt_tokens":12,"completion_tokens":3}}`,
			`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
		)
	}))
	defer server.Close()

	t.Setenv(deepSeekAPIKeyEnv, "sk-test")
	t.Setenv(deepSeekAPIURLEnv, server.URL+"/chat/completions")
	t.Setenv(deepSeekFunctionCallingEnv, "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "deepseek-v4-flash",
		CatalogModel: "deepseek-v4-flash",
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
	if !ok || function["name"] != deepSeekDiagnosticSmokeToolName {
		t.Fatalf("ToolChoice = %#v, want diagnostic tool", received.ToolChoice)
	}
	check, ok := deepSeekDiagnosticCheckByName(report, "tool_smoke")
	if !ok || check.Status != DiagnosticStatusOK {
		t.Fatalf("tool_smoke check = %#v, %v; want ok", check, ok)
	}
}

func TestDiagnoseDeepSeek_ToolSmokeFailsWithoutToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeDeepSeekSSE(w, `{"choices":[{"delta":{"content":"plain text"}}]}`)
	}))
	defer server.Close()

	t.Setenv(deepSeekAPIKeyEnv, "sk-test")
	t.Setenv(deepSeekAPIURLEnv, server.URL+"/chat/completions")
	t.Setenv(deepSeekFunctionCallingEnv, "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "deepseek-v4-flash",
		CatalogModel: "deepseek-v4-flash",
		RunSmoke:     true,
		ToolSmoke:    true,
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want tool smoke failure: %#v", report.Checks)
	}
	check, ok := deepSeekDiagnosticCheckByName(report, "tool_smoke")
	if !ok || check.Status != DiagnosticStatusFail {
		t.Fatalf("tool_smoke check = %#v, %v; want fail", check, ok)
	}
}

func TestDiagnoseDeepSeek_FunctionCallingDisabledSkipsToolAndRunsTextFallback(t *testing.T) {
	requests := 0
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeDeepSeekSSE(w,
			`{"choices":[{"delta":{"content":"fallback ok"}}]}`,
			`{"choices":[],"usage":{"prompt_tokens":8,"completion_tokens":2}}`,
		)
	}))
	defer server.Close()

	t.Setenv(deepSeekAPIKeyEnv, "sk-test")
	t.Setenv(deepSeekAPIURLEnv, server.URL+"/chat/completions")
	t.Setenv(deepSeekFunctionCallingEnv, "0")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "deepseek-v4-flash",
		CatalogModel: "deepseek-v4-flash",
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
	check, ok := deepSeekDiagnosticCheckByName(report, "tool_smoke")
	if !ok || check.Status != DiagnosticStatusWarn {
		t.Fatalf("tool_smoke check = %#v, %v; want warn skip", check, ok)
	}
	if report.Smoke == nil || len(report.Smoke.Requests) != 2 || !report.Smoke.Requests[1].Skipped {
		t.Fatalf("Smoke requests = %#v, want text fallback plus skipped tool", report.Smoke)
	}
}

func writeDeepSeekSSE(w http.ResponseWriter, chunks ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	for _, chunk := range chunks {
		_, _ = w.Write([]byte("data: " + chunk + "\n\n"))
	}
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
}

func deepSeekDiagnosticCheckByName(report DiagnosticReport, name string) (DiagnosticCheck, bool) {
	for _, check := range report.Checks {
		if check.Name == name {
			return check, true
		}
	}
	return DiagnosticCheck{}, false
}

func decodeDeepSeekDiagnosticPreviewBody(t *testing.T, body any) map[string]any {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal preview body: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode preview body: %v\n%s", err, string(payload))
	}
	return decoded
}
