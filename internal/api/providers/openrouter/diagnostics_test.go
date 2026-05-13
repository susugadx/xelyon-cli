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

func TestDiagnoseOpenRouter_MissingAPIKeyFails(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "")
	t.Setenv(openRouterAPIURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want missing API key failure: %#v", report.Checks)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "auth", DiagnosticStatusFail)
}

func TestDiagnoseOpenRouter_DefaultModelUsesAnthropicMessagesRoute(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, "")
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})

	if report.Provider != "openrouter" {
		t.Fatalf("Provider = %q, want openrouter", report.Provider)
	}
	if report.Model != "anthropic/claude-sonnet-4.6" {
		t.Fatalf("Model = %q, want default Claude model", report.Model)
	}
	if report.CatalogModel != "anthropic/claude-sonnet-4.6" {
		t.Fatalf("CatalogModel = %q, want default catalog model", report.CatalogModel)
	}
	if report.Route != DiagnosticRouteAnthropicMessages {
		t.Fatalf("Route = %q, want anthropic_messages", report.Route)
	}
	if !strings.HasSuffix(report.APIURL, "/v1/messages") {
		t.Fatalf("APIURL = %q, want Anthropic Skin messages endpoint", report.APIURL)
	}
	if report.UpstreamProvider != "anthropic" || report.UpstreamModel != "claude-sonnet-4.6" {
		t.Fatalf("upstream = %s/%s, want anthropic/claude-sonnet-4.6", report.UpstreamProvider, report.UpstreamModel)
	}
	if report.MaxOutputTokens != 64000 || report.ContextWindowTokens != 200000 {
		t.Fatalf("token policy = max %d context %d, want Claude metadata", report.MaxOutputTokens, report.ContextWindowTokens)
	}
	for _, name := range []string{"auth", "endpoint", "provider_registration", "model", "catalog_model", "route", "catalog_policy", "function_calling", "image_input"} {
		requireOpenRouterDiagnosticCheckStatus(t, report, name, DiagnosticStatusOK)
	}
}

func TestDiagnoseOpenRouter_OpenAIModelUsesChatCompletionsRoute(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "corp-openrouter-gpt",
		CatalogModel: "openai/gpt-5.4",
	})

	if report.Model != "corp-openrouter-gpt" || report.ModelSource != "--model" {
		t.Fatalf("model = %q (%s), want explicit model", report.Model, report.ModelSource)
	}
	if report.CatalogModel != "openai/gpt-5.4" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit catalog model", report.CatalogModel, report.CatalogModelSource)
	}
	if report.Route != DiagnosticRouteChatCompletions {
		t.Fatalf("Route = %q, want chat_completions", report.Route)
	}
	if report.UpstreamProvider != "openai" || report.UpstreamModel != "gpt-5.4" {
		t.Fatalf("upstream = %s/%s, want openai/gpt-5.4", report.UpstreamProvider, report.UpstreamModel)
	}
	catalogPolicy := requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusOK)
	if !strings.Contains(catalogPolicy.Detail, "context_window=1000000") ||
		!strings.Contains(catalogPolicy.Detail, "max_output_tokens=64000") ||
		!strings.Contains(catalogPolicy.Detail, "pricing=input $2.50/M") {
		t.Fatalf("catalog_policy detail = %q, want OpenRouter delegated policy detail", catalogPolicy.Detail)
	}
}

func TestDiagnoseOpenRouter_AliasCatalogClaudeWarnsRouteUsesRequestModel(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, "")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openrouter", config.ProviderModelConfig{
		DefaultModel: "corp-claude-alias",
		CatalogModel: "anthropic/claude-sonnet-4.6",
	})

	report := Diagnose(context.Background(), DiagnosticOptions{Config: cfg})
	if report.Route != DiagnosticRouteChatCompletions {
		t.Fatalf("Route = %q, want alias to follow request model and use chat_completions", report.Route)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "route", DiagnosticStatusWarn)
}

func TestDiagnoseOpenRouter_NonOpenRouterCatalogModelDoesNotUseGlobalMetadata(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "corp-openrouter-model",
		CatalogModel: "gpt-5.5",
	})

	if report.CatalogModel != "gpt-5.5" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit value in report", report.CatalogModel, report.CatalogModelSource)
	}
	if report.ContextWindowTokens != 0 {
		t.Fatalf("ContextWindowTokens = %d, want non-OpenRouter catalog metadata ignored", report.ContextWindowTokens)
	}
	if report.MaxOutputTokens == 128000 {
		t.Fatalf("MaxOutputTokens = %d, want OpenAI catalog max output ignored", report.MaxOutputTokens)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_model", DiagnosticStatusWarn)
	catalogPolicy := requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusWarn)
	if !strings.Contains(catalogPolicy.Detail, "context_window=unknown") ||
		!strings.Contains(catalogPolicy.Detail, "max_output_tokens=unknown") ||
		strings.Contains(catalogPolicy.Detail, "1050000") ||
		strings.Contains(catalogPolicy.Detail, "128000") {
		t.Fatalf("catalog_policy detail = %q, want no OpenAI token metadata", catalogPolicy.Detail)
	}
}

func TestDiagnoseOpenRouter_InvalidExplicitCatalogClearsConfiguredCatalogMetadata(t *testing.T) {
	var received openaicompat.ChatCompletionsRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeOpenRouterChatCompletionsSSE(w,
			`{"choices":[{"delta":{"content":"fallback ok"}}]}`,
			`{"choices":[{"delta":{}}],"usage":{"prompt_tokens":1000,"completion_tokens":2}}`,
		)
	}))
	defer server.Close()

	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, server.URL)

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openrouter", config.ProviderModelConfig{
		DefaultModel: "corp-openrouter-model",
		CatalogModel: "openai/gpt-5.5",
	})

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       cfg,
		CatalogModel: "bogus",
		RunSmoke:     true,
		TextSmoke:    true,
	})

	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	if report.Model != "corp-openrouter-model" || report.CatalogModel != "bogus" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("model/catalog = %q/%q (%s), want configured model with explicit invalid catalog", report.Model, report.CatalogModel, report.CatalogModelSource)
	}
	if received.Model != "corp-openrouter-model" {
		t.Fatalf("smoke model = %q, want configured request model", received.Model)
	}
	if report.ContextWindowTokens != 0 {
		t.Fatalf("ContextWindowTokens = %d, want invalid catalog metadata ignored", report.ContextWindowTokens)
	}
	if report.MaxOutputTokens != 64000 {
		t.Fatalf("MaxOutputTokens = %d, want OpenRouter provider fallback without stale catalog metadata", report.MaxOutputTokens)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_model", DiagnosticStatusWarn)
	catalogPolicy := requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusWarn)
	if !strings.Contains(catalogPolicy.Detail, "catalog_model=bogus") ||
		strings.Contains(catalogPolicy.Detail, "1050000") ||
		strings.Contains(catalogPolicy.Detail, "128000") {
		t.Fatalf("catalog_policy detail = %q, want invalid catalog only and no stale OpenAI metadata", catalogPolicy.Detail)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "cost", DiagnosticStatusWarn)
	if report.Smoke == nil || !report.Smoke.Cost.PricingUnavailable || report.Smoke.Cost.USD != 0 {
		t.Fatalf("Smoke cost = %#v, want pricing unavailable without stale configured catalog", report.Smoke)
	}
}

func TestDiagnoseOpenRouter_RoutedModelWarnsMismatchedCatalogModel(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "anthropic/claude-sonnet-4.6",
		CatalogModel: "openai/gpt-5.4",
	})

	if report.CatalogModel != "openai/gpt-5.4" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit value in report", report.CatalogModel, report.CatalogModelSource)
	}
	if report.UpstreamProvider != "anthropic" || report.UpstreamModel != "claude-sonnet-4.6" {
		t.Fatalf("upstream = %s/%s, want actual routed request model", report.UpstreamProvider, report.UpstreamModel)
	}
	if report.ContextWindowTokens != 200000 || report.MaxOutputTokens != 64000 {
		t.Fatalf("token policy = max %d context %d, want request model metadata", report.MaxOutputTokens, report.ContextWindowTokens)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_model", DiagnosticStatusWarn)
	catalogPolicy := requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusWarn)
	if !strings.Contains(catalogPolicy.Detail, "requested_catalog_model=openai/gpt-5.4") ||
		!strings.Contains(catalogPolicy.Detail, "policy_catalog_model=anthropic/claude-sonnet-4.6") ||
		!strings.Contains(catalogPolicy.Detail, "context_window=200000") ||
		!strings.Contains(catalogPolicy.Detail, "pricing=input $3.00/M") ||
		strings.Contains(catalogPolicy.Detail, "pricing=input $2.50/M") {
		t.Fatalf("catalog_policy detail = %q, want request model policy and no OpenAI pricing", catalogPolicy.Detail)
	}
}

func TestDiagnoseOpenRouter_RoutedUnknownModelRejectsCrossOwnerCatalogModel(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "anthropic/claude-future-prod",
		CatalogModel: "openai/gpt-5.4",
	})

	if report.UpstreamProvider != "anthropic" || report.UpstreamModel != "claude-future-prod" {
		t.Fatalf("upstream = %s/%s, want request model owner", report.UpstreamProvider, report.UpstreamModel)
	}
	if report.ContextWindowTokens != 0 {
		t.Fatalf("ContextWindowTokens = %d, want cross-owner catalog metadata ignored", report.ContextWindowTokens)
	}
	if report.MaxOutputTokens == 32768 {
		t.Fatalf("MaxOutputTokens = %d, want OpenAI catalog max output ignored", report.MaxOutputTokens)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_model", DiagnosticStatusWarn)
	catalogPolicy := requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusWarn)
	if !strings.Contains(catalogPolicy.Detail, "model owner=anthropic but catalog_model owner=openai") ||
		strings.Contains(catalogPolicy.Detail, "pricing=input $2.50/M") ||
		strings.Contains(catalogPolicy.Detail, "context_window=1000000") {
		t.Fatalf("catalog_policy detail = %q, want cross-owner catalog ignored", catalogPolicy.Detail)
	}
}

func TestDiagnoseOpenRouter_PrintRequestDoesNotRequireAPIKeyOrSendNetwork(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Fatalf("unexpected network request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	t.Setenv(openRouterAPIKeyEnv, "")
	t.Setenv(openRouterAPIURLEnv, server.URL+"/v1/chat/completions")
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "anthropic/claude-sonnet-4.6",
		CatalogModel: "anthropic/claude-sonnet-4.6",
		ToolSmoke:    true,
		PrintRequest: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want print request without API key to pass: %#v", report.Checks)
	}
	requireOpenRouterDiagnosticCheckAbsent(t, report, "auth")
	if report.Smoke != nil {
		t.Fatalf("Smoke = %#v, want nil for --print-request", report.Smoke)
	}
	if requests != 0 {
		t.Fatalf("network requests = %d, want 0", requests)
	}
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 1 {
		t.Fatalf("RequestPreview = %#v, want one tool request", report.RequestPreview)
	}
	preview := report.RequestPreview.Requests[0]
	if preview.Name != "tool" || !preview.ToolPayload || preview.Route != DiagnosticRouteAnthropicMessages || preview.URL != server.URL+"/v1/messages" {
		t.Fatalf("preview = %#v, want tool request to Anthropic Skin endpoint", preview)
	}
	if preview.Headers["Authorization"] != "Bearer <redacted>" || preview.Headers["X-Title"] != "XELYON CLI" {
		t.Fatalf("headers = %#v, want redacted OpenRouter headers", preview.Headers)
	}
	body := decodeOpenRouterDiagnosticPreviewBodyForTest(t, preview.Body)
	if body["model"] != "anthropic/claude-sonnet-4.6" || body["anthropic_version"] == "" || body["max_tokens"] != float64(64) {
		t.Fatalf("preview body = %#v, want Anthropic Skin diagnostic body", body)
	}
	tools, ok := body["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v, want diagnostic Claude tool", body["tools"])
	}
	toolChoice, ok := body["tool_choice"].(map[string]any)
	if !ok {
		t.Fatalf("tool_choice = %T, want forced Anthropic Skin tool choice", body["tool_choice"])
	}
	if toolChoice["type"] != "tool" || toolChoice["name"] != openRouterDiagnosticSmokeToolName {
		t.Fatalf("tool_choice = %#v, want diagnostic tool choice", toolChoice)
	}
}

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

func writeOpenRouterChatCompletionsSSE(w http.ResponseWriter, chunks ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	for _, chunk := range chunks {
		_, _ = w.Write([]byte("data: " + chunk + "\n\n"))
	}
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
}

func writeOpenRouterAnthropicSSE(w http.ResponseWriter, chunks ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	for _, chunk := range chunks {
		_, _ = w.Write([]byte("data: " + chunk + "\n\n"))
	}
}

func requireOpenRouterDiagnosticCheckStatus(t *testing.T, report DiagnosticReport, name string, status DiagnosticStatus) DiagnosticCheck {
	t.Helper()

	check, ok := openRouterDiagnosticCheckByName(report, name)
	if !ok {
		t.Fatalf("%s check missing: %#v", name, report.Checks)
	}
	if check.Status != status {
		t.Fatalf("%s check = %#v; want %s", name, check, status)
	}
	return check
}

func requireOpenRouterDiagnosticCheckAbsent(t *testing.T, report DiagnosticReport, name string) {
	t.Helper()

	if check, ok := openRouterDiagnosticCheckByName(report, name); ok {
		t.Fatalf("%s check was added unexpectedly: %#v", name, check)
	}
}

func openRouterDiagnosticCheckByName(report DiagnosticReport, name string) (DiagnosticCheck, bool) {
	for _, check := range report.Checks {
		if check.Name == name {
			return check, true
		}
	}
	return DiagnosticCheck{}, false
}

func decodeOpenRouterDiagnosticPreviewBodyForTest(t *testing.T, body any) map[string]any {
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
