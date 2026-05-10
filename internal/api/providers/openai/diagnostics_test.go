package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseOpenAI_MissingAPIKeyFails(t *testing.T) {
	t.Setenv(openAIAPIKeyEnv, "")
	t.Setenv(openAIAPIURLEnv, "")
	t.Setenv(openAIResponsesURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want missing API key failure: %#v", report.Checks)
	}
	check, ok := openAIDiagnosticCheckByName(report, "auth")
	if !ok || check.Status != DiagnosticStatusFail {
		t.Fatalf("auth check = %#v, %v; want fail", check, ok)
	}
}

func TestDiagnoseOpenAI_InvalidActiveResponsesURLFails(t *testing.T) {
	t.Setenv(openAIAPIKeyEnv, "sk-test")
	t.Setenv(openAIAPIURLEnv, "")
	t.Setenv(openAIResponsesURLEnv, "://bad")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config: config.DefaultConfig(),
		Model:  "gpt-5.4",
	})
	check, ok := openAIDiagnosticCheckByName(report, "responses_url")
	if !ok || check.Status != DiagnosticStatusFail {
		t.Fatalf("responses_url check = %#v, %v; want fail", check, ok)
	}
}

func TestDiagnoseOpenAI_ModelPolicyRouteAndFunctionCalling(t *testing.T) {
	t.Setenv(openAIAPIKeyEnv, "sk-test")
	t.Setenv(openAIAPIURLEnv, "")
	t.Setenv(openAIResponsesURLEnv, "")
	t.Setenv("OPENAI_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "corp-openai-deployment",
		CatalogModel: "gpt-5.4",
	})

	if report.Provider != "openai" {
		t.Fatalf("Provider = %q, want openai", report.Provider)
	}
	if report.Model != "corp-openai-deployment" || report.ModelSource != "--model" {
		t.Fatalf("model = %q (%s), want explicit model", report.Model, report.ModelSource)
	}
	if report.CatalogModel != "gpt-5.4" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit catalog model", report.CatalogModel, report.CatalogModelSource)
	}
	if report.Route != DiagnosticRouteResponsesStreaming {
		t.Fatalf("Route = %q, want responses streaming", report.Route)
	}
	if report.MaxOutputTokens != 16384 {
		t.Fatalf("MaxOutputTokens = %d, want OpenAI provider default max output", report.MaxOutputTokens)
	}
	if report.ContextWindowTokens != 1000000 {
		t.Fatalf("ContextWindowTokens = %d, want gpt-5.4 context window", report.ContextWindowTokens)
	}
	if !report.FunctionCallingEnabled {
		t.Fatal("FunctionCallingEnabled = false, want true")
	}
	for _, name := range []string{"provider_registration", "model", "route", "catalog_policy", "function_calling", "responses_retention"} {
		check, ok := openAIDiagnosticCheckByName(report, name)
		if !ok || check.Status != DiagnosticStatusOK {
			t.Fatalf("%s check = %#v, %v; want ok", name, check, ok)
		}
	}
}

func TestDiagnoseOpenAI_FunctionCallingDisabled(t *testing.T) {
	t.Setenv(openAIAPIKeyEnv, "sk-test")
	t.Setenv(openAIAPIURLEnv, "")
	t.Setenv(openAIResponsesURLEnv, "")
	t.Setenv("OPENAI_FUNCTION_CALLING", "0")

	report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
	if report.FunctionCallingEnabled {
		t.Fatal("FunctionCallingEnabled = true, want false")
	}
	check, ok := openAIDiagnosticCheckByName(report, "function_calling")
	if !ok || check.Status != DiagnosticStatusOK || !strings.Contains(check.Detail, "OPENAI_FUNCTION_CALLING=0") {
		t.Fatalf("function_calling check = %#v, %v; want disabled detail", check, ok)
	}
}

func TestDiagnoseOpenAI_ResponsesNonStreamingSmokeObservesResponseIDUsageCostAndRequestShape(t *testing.T) {
	var received struct {
		Path string
		Body map[string]any
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&received.Body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Fatalf("Authorization = %q, want Bearer sk-test", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_doctor","output_text":"xelyon openai doctor ok","usage":{"input_tokens":10,"output_tokens":6,"input_tokens_details":{"cached_tokens":4},"output_tokens_details":{"reasoning_tokens":2}}}`))
	}))
	defer server.Close()

	t.Setenv(openAIAPIKeyEnv, "sk-test")
	t.Setenv(openAIAPIURLEnv, "")
	t.Setenv(openAIResponsesURLEnv, server.URL)

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gpt-5.5-pro",
		CatalogModel: "gpt-5.5-pro",
		RunSmoke:     true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	if report.Route != DiagnosticRouteResponsesNonStreaming {
		t.Fatalf("Route = %q, want non-streaming Responses", report.Route)
	}
	if report.Smoke == nil || report.Smoke.ResponseID != "resp_doctor" {
		t.Fatalf("Smoke = %#v, want response ID", report.Smoke)
	}
	if !report.Smoke.UsageObserved || report.Smoke.Usage.InputTokens != 10 || report.Smoke.Usage.OutputTokens != 4 || report.Smoke.Usage.CachedInputTokens != 4 || report.Smoke.Usage.ThinkingTokens != 2 {
		t.Fatalf("Smoke usage = %+v observed=%t, want normalized usage", report.Smoke.Usage, report.Smoke.UsageObserved)
	}
	if report.Smoke.Cost.PricingUnavailable || report.Smoke.Cost.USD <= 0 {
		t.Fatalf("Smoke cost = %+v, want available positive estimate", report.Smoke.Cost)
	}
	if received.Path != "/" {
		t.Fatalf("path = %q, want test server root path", received.Path)
	}
	if received.Body["model"] != "gpt-5.5-pro" {
		t.Fatalf("model = %#v, want gpt-5.5-pro", received.Body["model"])
	}
	if received.Body["stream"] == true {
		t.Fatalf("stream = true, want omitted/false for gpt-5.5-pro")
	}
	if received.Body["store"] != false {
		t.Fatalf("store = %#v, want false for doctor smoke", received.Body["store"])
	}
	if received.Body["max_output_tokens"] != float64(defaultOpenAIDiagnosticSmokeMaxOutputToks) {
		t.Fatalf("max_output_tokens = %#v, want smoke override", received.Body["max_output_tokens"])
	}
	if _, ok := received.Body["tools"]; ok {
		t.Fatalf("tools should be omitted in text smoke: %#v", received.Body["tools"])
	}
}

func TestDiagnoseOpenAI_ToolSmokeSendsForcedResponsesToolPayload(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_tool_doctor","output":[{"type":"function_call","call_id":"call_probe","name":"xelyon_openai_doctor_probe","arguments":"{\"value\":\"openai-tool-ok\"}"}],"usage":{"input_tokens":8,"output_tokens":4,"output_tokens_details":{"reasoning_tokens":1}}}`))
	}))
	defer server.Close()

	t.Setenv(openAIAPIKeyEnv, "sk-test")
	t.Setenv(openAIAPIURLEnv, "")
	t.Setenv(openAIResponsesURLEnv, server.URL)
	t.Setenv("OPENAI_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gpt-5.5-pro",
		CatalogModel: "gpt-5.5-pro",
		RunSmoke:     true,
		ToolSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	if report.Smoke == nil || !report.Smoke.ToolPayload {
		t.Fatalf("Smoke = %#v, want tool payload", report.Smoke)
	}
	if !strings.Contains(report.Smoke.Content, `"tool":"xelyon_openai_doctor_probe"`) {
		t.Fatalf("Smoke.Content = %q, want diagnostic tool JSON", report.Smoke.Content)
	}
	tools, ok := received["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v, want one diagnostic tool", received["tools"])
	}
	tool, ok := tools[0].(map[string]any)
	if !ok || tool["name"] != "xelyon_openai_doctor_probe" {
		t.Fatalf("tool = %#v, want diagnostic probe", tools[0])
	}
	toolChoice, ok := received["tool_choice"].(map[string]any)
	if !ok || toolChoice["type"] != "function" || toolChoice["name"] != "xelyon_openai_doctor_probe" {
		t.Fatalf("tool_choice = %#v, want forced diagnostic function", received["tool_choice"])
	}
}

func TestDiagnoseOpenAI_TextAndToolSmokeRunSeparateRequests(t *testing.T) {
	var received []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		received = append(received, body)
		w.Header().Set("Content-Type", "application/json")
		switch len(received) {
		case 1:
			_, _ = w.Write([]byte(`{"id":"resp_text_doctor","output_text":"xelyon openai doctor ok","usage":{"input_tokens":5,"output_tokens":3}}`))
		case 2:
			_, _ = w.Write([]byte(`{"id":"resp_tool_doctor","output":[{"type":"function_call","call_id":"call_probe","name":"xelyon_openai_doctor_probe","arguments":"{\"value\":\"openai-tool-ok\"}"}],"usage":{"input_tokens":8,"output_tokens":4,"output_tokens_details":{"reasoning_tokens":1}}}`))
		default:
			t.Fatalf("unexpected smoke request count %d", len(received))
		}
	}))
	defer server.Close()

	t.Setenv(openAIAPIKeyEnv, "sk-test")
	t.Setenv(openAIAPIURLEnv, "")
	t.Setenv(openAIResponsesURLEnv, server.URL)
	t.Setenv("OPENAI_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gpt-5.5-pro",
		CatalogModel: "gpt-5.5-pro",
		RunSmoke:     true,
		TextSmoke:    true,
		ToolSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	if len(received) != 2 {
		t.Fatalf("received %d smoke requests, want text and tool requests", len(received))
	}
	if _, ok := received[0]["tools"]; ok {
		t.Fatalf("text smoke tools = %#v, want omitted", received[0]["tools"])
	}
	if _, ok := received[0]["tool_choice"]; ok {
		t.Fatalf("text smoke tool_choice = %#v, want omitted", received[0]["tool_choice"])
	}
	if _, ok := received[1]["tools"].([]any); !ok {
		t.Fatalf("tool smoke tools = %#v, want diagnostic tools", received[1]["tools"])
	}
	if _, ok := received[1]["tool_choice"].(map[string]any); !ok {
		t.Fatalf("tool smoke tool_choice = %#v, want forced diagnostic function", received[1]["tool_choice"])
	}
	if report.Smoke == nil || len(report.Smoke.Requests) != 2 {
		t.Fatalf("Smoke = %#v, want two request results", report.Smoke)
	}
	if report.Smoke.Requests[0].Name != "text" || report.Smoke.Requests[0].ToolPayload {
		t.Fatalf("text request = %#v, want text request result", report.Smoke.Requests[0])
	}
	if report.Smoke.Requests[1].Name != "tool" || !report.Smoke.Requests[1].ToolPayload {
		t.Fatalf("tool request = %#v, want tool request result", report.Smoke.Requests[1])
	}
	if !strings.Contains(report.Smoke.Requests[1].Content, `"tool":"xelyon_openai_doctor_probe"`) {
		t.Fatalf("tool content = %q, want diagnostic tool JSON", report.Smoke.Requests[1].Content)
	}
	if !report.Smoke.UsageObserved || report.Smoke.Usage.InputTokens != 13 || report.Smoke.Usage.OutputTokens != 6 || report.Smoke.Usage.ThinkingTokens != 1 {
		t.Fatalf("Smoke usage = %+v observed=%t, want aggregate usage", report.Smoke.Usage, report.Smoke.UsageObserved)
	}
	if !report.Smoke.ToolPayload {
		t.Fatal("Smoke.ToolPayload = false, want aggregate tool payload observation")
	}
}

func TestDiagnoseOpenAI_ToolSmokeSkippedWhenFunctionCallingDisabled(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_text","output_text":"xelyon openai doctor ok","usage":{"input_tokens":5,"output_tokens":3}}`))
	}))
	defer server.Close()

	t.Setenv(openAIAPIKeyEnv, "sk-test")
	t.Setenv(openAIAPIURLEnv, "")
	t.Setenv(openAIResponsesURLEnv, server.URL)
	t.Setenv("OPENAI_FUNCTION_CALLING", "0")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gpt-5.5-pro",
		CatalogModel: "gpt-5.5-pro",
		RunSmoke:     true,
		ToolSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	check, ok := openAIDiagnosticCheckByName(report, "tool_smoke")
	if !ok || check.Status != DiagnosticStatusWarn {
		t.Fatalf("tool_smoke check = %#v, %v; want warn skip", check, ok)
	}
	if report.Smoke == nil || report.Smoke.ToolPayload {
		t.Fatalf("Smoke = %#v, want text smoke without tool payload", report.Smoke)
	}
	if _, ok := received["tools"]; ok {
		t.Fatalf("tools should be omitted when function calling is disabled: %#v", received)
	}
	if _, ok := received["tool_choice"]; ok {
		t.Fatalf("tool_choice should be omitted when function calling is disabled: %#v", received)
	}
}

func TestDiagnoseOpenAI_ChatCompletionsSmokeUsesCompatRouteWithoutResponseID(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"xelyon openai doctor ok"}}]}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":7,"completion_tokens":4,"prompt_tokens_details":{"cached_tokens":2}}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: [DONE]`)
		fmt.Fprintln(w)
	}))
	defer server.Close()

	t.Setenv(openAIAPIKeyEnv, "sk-test")
	t.Setenv(openAIAPIURLEnv, server.URL)
	t.Setenv(openAIResponsesURLEnv, "")
	t.Setenv("OPENAI_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gpt-4",
		CatalogModel: "gpt-4",
		RunSmoke:     true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	if report.Route != DiagnosticRouteChatCompletions {
		t.Fatalf("Route = %q, want Chat Completions", report.Route)
	}
	if report.Smoke == nil || report.Smoke.ResponseID != "" {
		t.Fatalf("Smoke = %#v, want no response ID for Chat Completions", report.Smoke)
	}
	if !report.Smoke.UsageObserved || report.Smoke.Usage.InputTokens != 7 || report.Smoke.Usage.OutputTokens != 4 || report.Smoke.Usage.CachedInputTokens != 2 {
		t.Fatalf("Smoke usage = %+v observed=%t, want stream usage", report.Smoke.Usage, report.Smoke.UsageObserved)
	}
	if received["model"] != "gpt-4" {
		t.Fatalf("model = %#v, want gpt-4", received["model"])
	}
	if received["stream"] != true {
		t.Fatalf("stream = %#v, want true", received["stream"])
	}
	if _, ok := received["tools"]; ok {
		t.Fatalf("tools should be omitted in text smoke: %#v", received["tools"])
	}
	if _, ok := received["tool_choice"]; ok {
		t.Fatalf("tool_choice should be omitted in text smoke: %#v", received["tool_choice"])
	}
}

func openAIDiagnosticCheckByName(report DiagnosticReport, name string) (DiagnosticCheck, bool) {
	for _, check := range report.Checks {
		if check.Name == name {
			return check, true
		}
	}
	return DiagnosticCheck{}, false
}
