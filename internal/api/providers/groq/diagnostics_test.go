package groq

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseGroq_MissingAPIKeyFails(t *testing.T) {
	t.Setenv(groqAPIKeyEnv, "")
	t.Setenv(groqAPIURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want missing API key failure: %#v", report.Checks)
	}
	check, ok := groqDiagnosticCheckByName(report, "auth")
	if !ok || check.Status != DiagnosticStatusFail {
		t.Fatalf("auth check = %#v, %v; want fail", check, ok)
	}
}

func TestDiagnoseGroq_ModelCatalogPolicyRouteAndFunctionCalling(t *testing.T) {
	t.Setenv(groqAPIKeyEnv, "gsk-test")
	t.Setenv(groqAPIURLEnv, "")
	t.Setenv(groqFunctionCallingEnv, "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "corp-groq-model",
		CatalogModel: "meta-llama/llama-4-scout-17b-16e-instruct",
	})

	if report.Provider != "groq" {
		t.Fatalf("Provider = %q, want groq", report.Provider)
	}
	if report.Model != "corp-groq-model" || report.ModelSource != "--model" {
		t.Fatalf("model = %q (%s), want explicit model", report.Model, report.ModelSource)
	}
	if report.CatalogModel != "meta-llama/llama-4-scout-17b-16e-instruct" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit catalog model", report.CatalogModel, report.CatalogModelSource)
	}
	if report.Route != DiagnosticRouteChatCompletions {
		t.Fatalf("Route = %q, want chat_completions", report.Route)
	}
	if report.MaxOutputTokens != 8192 {
		t.Fatalf("MaxOutputTokens = %d, want Groq catalog max output", report.MaxOutputTokens)
	}
	if report.ContextWindowTokens != 131072 {
		t.Fatalf("ContextWindowTokens = %d, want Llama 4 Scout context window", report.ContextWindowTokens)
	}
	if !report.FunctionCallingEnabled {
		t.Fatal("FunctionCallingEnabled = false, want true")
	}
	for _, name := range []string{"auth", "endpoint", "provider_registration", "model", "catalog_model", "route", "catalog_policy", "function_calling"} {
		check, ok := groqDiagnosticCheckByName(report, name)
		if !ok || check.Status != DiagnosticStatusOK {
			t.Fatalf("%s check = %#v, %v; want ok", name, check, ok)
		}
	}
	catalogPolicy, _ := groqDiagnosticCheckByName(report, "catalog_policy")
	if !strings.Contains(catalogPolicy.Detail, "context_window=131072") ||
		!strings.Contains(catalogPolicy.Detail, "max_output_tokens=8192") ||
		!strings.Contains(catalogPolicy.Detail, "pricing=input $0.11/M") {
		t.Fatalf("catalog_policy detail = %q, want Groq policy detail", catalogPolicy.Detail)
	}
}

func TestDiagnoseGroq_NonGroqCatalogModelDoesNotUseGlobalMetadata(t *testing.T) {
	t.Setenv(groqAPIKeyEnv, "gsk-test")
	t.Setenv(groqAPIURLEnv, "")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("groq", config.ProviderModelConfig{
		DefaultModel: "corp-groq-model",
		CatalogModel: "meta-llama/llama-4-scout-17b-16e-instruct",
	})

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       cfg,
		Model:        "corp-groq-model",
		CatalogModel: "gpt-5.5",
	})

	if report.CatalogModel != "gpt-5.5" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit value in report", report.CatalogModel, report.CatalogModelSource)
	}
	if report.ContextWindowTokens != 0 {
		t.Fatalf("ContextWindowTokens = %d, want non-Groq catalog metadata ignored", report.ContextWindowTokens)
	}
	if report.MaxOutputTokens == 128000 {
		t.Fatalf("MaxOutputTokens = %d, want OpenAI catalog max output ignored", report.MaxOutputTokens)
	}
	catalogModel, ok := groqDiagnosticCheckByName(report, "catalog_model")
	if !ok || catalogModel.Status != DiagnosticStatusWarn {
		t.Fatalf("catalog_model check = %#v, %v; want warn", catalogModel, ok)
	}
	catalogPolicy, ok := groqDiagnosticCheckByName(report, "catalog_policy")
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

func TestDiagnoseGroq_PrintRequestDoesNotRequireAPIKeyOrSendNetwork(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Fatalf("unexpected network request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	t.Setenv(groqAPIKeyEnv, "")
	t.Setenv(groqAPIURLEnv, server.URL+"/openai/v1/chat/completions")
	t.Setenv(groqFunctionCallingEnv, "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "meta-llama/llama-4-scout-17b-16e-instruct",
		CatalogModel: "meta-llama/llama-4-scout-17b-16e-instruct",
		ToolSmoke:    true,
		PrintRequest: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want print request without API key to pass: %#v", report.Checks)
	}
	if _, ok := groqDiagnosticCheckByName(report, "auth"); ok {
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
	if preview.Name != "tool" || !preview.ToolPayload || preview.URL != server.URL+"/openai/v1/chat/completions" {
		t.Fatalf("preview = %#v, want tool request to Groq endpoint", preview)
	}
	if preview.Headers["Authorization"] != "Bearer <redacted>" {
		t.Fatalf("Authorization preview = %q, want redacted bearer", preview.Headers["Authorization"])
	}
	body := decodeGroqDiagnosticPreviewBody(t, preview.Body)
	if body.Model != "meta-llama/llama-4-scout-17b-16e-instruct" || len(body.Tools) != 1 || body.ToolChoice == nil {
		t.Fatalf("preview body = %#v, want forced diagnostic tool payload", body)
	}
}

func TestDiagnoseGroq_PrintRequestSkipsDisabledToolPreview(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Fatalf("unexpected network request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	t.Setenv(groqAPIKeyEnv, "")
	t.Setenv(groqAPIURLEnv, server.URL+"/openai/v1/chat/completions")
	t.Setenv(groqFunctionCallingEnv, "0")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "meta-llama/llama-4-scout-17b-16e-instruct",
		CatalogModel: "meta-llama/llama-4-scout-17b-16e-instruct",
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
	if tool.Name != "tool" || !tool.Skipped || !tool.ToolPayload || !strings.Contains(tool.SkipReason, groqFunctionCallingEnv+"=0") {
		t.Fatalf("tool preview = %#v, want skipped disabled tool request", tool)
	}
}

func TestDiagnoseGroq_TextSmokeObservesUsageAndCost(t *testing.T) {
	var received openaicompat.ChatCompletionsRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer gsk-test" {
			t.Fatalf("Authorization = %q, want Bearer gsk-test", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeGroqSSE(w,
			`{"choices":[{"delta":{"content":"xelyon groq doctor ok"}}]}`,
			`{"choices":[{"delta":{}}],"usage":{"prompt_tokens":10,"completion_tokens":6,"prompt_tokens_details":{"cached_tokens":2},"completion_tokens_details":{"reasoning_tokens":1}}}`,
		)
	}))
	defer server.Close()

	t.Setenv(groqAPIKeyEnv, "gsk-test")
	t.Setenv(groqAPIURLEnv, server.URL+"/openai/v1/chat/completions")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "meta-llama/llama-4-scout-17b-16e-instruct",
		CatalogModel: "meta-llama/llama-4-scout-17b-16e-instruct",
		RunSmoke:     true,
		TextSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	if received.Model != "meta-llama/llama-4-scout-17b-16e-instruct" || received.MaxTokens != defaultGroqDiagnosticSmokeMaxOutputTokens {
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
		check, ok := groqDiagnosticCheckByName(report, name)
		if !ok || check.Status != DiagnosticStatusOK {
			t.Fatalf("%s check = %#v, %v; want ok", name, check, ok)
		}
	}
}

func TestDiagnoseGroq_ToolSmokeRequiresToolCall(t *testing.T) {
	var received openaicompat.ChatCompletionsRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeGroqSSE(w,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"xelyon_groq_doctor_probe","arguments":"{\"value\":\"groq-tool-ok\"}"}}]}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":12,"completion_tokens":3}}`,
		)
	}))
	defer server.Close()

	t.Setenv(groqAPIKeyEnv, "gsk-test")
	t.Setenv(groqAPIURLEnv, server.URL+"/openai/v1/chat/completions")
	t.Setenv(groqFunctionCallingEnv, "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "meta-llama/llama-4-scout-17b-16e-instruct",
		CatalogModel: "meta-llama/llama-4-scout-17b-16e-instruct",
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
	if !ok || function["name"] != groqDiagnosticSmokeToolName {
		t.Fatalf("ToolChoice = %#v, want diagnostic tool", received.ToolChoice)
	}
	check, ok := groqDiagnosticCheckByName(report, "tool_smoke")
	if !ok || check.Status != DiagnosticStatusOK {
		t.Fatalf("tool_smoke check = %#v, %v; want ok", check, ok)
	}
}

func TestDiagnoseGroq_ToolSmokeFailsWithoutToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeGroqSSE(w, `{"choices":[{"delta":{"content":"plain text"}}]}`)
	}))
	defer server.Close()

	t.Setenv(groqAPIKeyEnv, "gsk-test")
	t.Setenv(groqAPIURLEnv, server.URL+"/openai/v1/chat/completions")
	t.Setenv(groqFunctionCallingEnv, "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "meta-llama/llama-4-scout-17b-16e-instruct",
		CatalogModel: "meta-llama/llama-4-scout-17b-16e-instruct",
		RunSmoke:     true,
		ToolSmoke:    true,
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want tool smoke failure: %#v", report.Checks)
	}
	check, ok := groqDiagnosticCheckByName(report, "tool_smoke")
	if !ok || check.Status != DiagnosticStatusFail {
		t.Fatalf("tool_smoke check = %#v, %v; want fail", check, ok)
	}
}

func TestDiagnoseGroq_FunctionCallingDisabledSkipsToolAndRunsTextFallback(t *testing.T) {
	requests := 0
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeGroqSSE(w,
			`{"choices":[{"delta":{"content":"fallback ok"}}]}`,
			`{"choices":[{"delta":{}}],"usage":{"prompt_tokens":8,"completion_tokens":2}}`,
		)
	}))
	defer server.Close()

	t.Setenv(groqAPIKeyEnv, "gsk-test")
	t.Setenv(groqAPIURLEnv, server.URL+"/openai/v1/chat/completions")
	t.Setenv(groqFunctionCallingEnv, "0")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "meta-llama/llama-4-scout-17b-16e-instruct",
		CatalogModel: "meta-llama/llama-4-scout-17b-16e-instruct",
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
	check, ok := groqDiagnosticCheckByName(report, "tool_smoke")
	if !ok || check.Status != DiagnosticStatusWarn {
		t.Fatalf("tool_smoke check = %#v, %v; want warn skip", check, ok)
	}
	if report.Smoke == nil || len(report.Smoke.Requests) != 2 || !report.Smoke.Requests[1].Skipped {
		t.Fatalf("Smoke requests = %#v, want text fallback plus skipped tool", report.Smoke)
	}
}

func writeGroqSSE(w http.ResponseWriter, chunks ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	for _, chunk := range chunks {
		_, _ = w.Write([]byte("data: " + chunk + "\n\n"))
	}
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
}

func groqDiagnosticCheckByName(report DiagnosticReport, name string) (DiagnosticCheck, bool) {
	for _, check := range report.Checks {
		if check.Name == name {
			return check, true
		}
	}
	return DiagnosticCheck{}, false
}

func decodeGroqDiagnosticPreviewBody(t *testing.T, body any) openaicompat.ChatCompletionsRequest {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal preview body: %v", err)
	}
	var decoded openaicompat.ChatCompletionsRequest
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode preview body: %v\n%s", err, string(payload))
	}
	return decoded
}
