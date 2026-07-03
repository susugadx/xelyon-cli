package openaisubscription

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestSubscriptionDiagnosticsWebSearchPrintRequestUsesNativePayloadShape(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())
	t.Setenv(subscriptionEndpointEnv, "https://user-secret:pass-secret@proxy.example.test/backend-api/codex/responses?token=query-secret#frag-secret")
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "high"

	report := DiagnoseOpenAISubscription(context.Background(), SubscriptionDiagnosticOptions{
		Config:         cfg,
		Model:          "gpt-5.5",
		CatalogModel:   "gpt-5.5",
		PrintRequest:   true,
		WebSearchSmoke: true,
	})

	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 1 {
		t.Fatalf("RequestPreview = %#v, want one web search preview", report.RequestPreview)
	}
	request := report.RequestPreview.Requests[0]
	if request.Name != "web_search" || !request.WebSearchPayload || request.ToolPayload {
		t.Fatalf("preview request = %+v, want web_search payload only", request)
	}
	if request.URL != report.Endpoint {
		t.Fatalf("preview URL = %q, want sanitized endpoint %q", request.URL, report.Endpoint)
	}
	body, ok := request.Body.(map[string]any)
	if !ok {
		t.Fatalf("preview body = %#v, want map", request.Body)
	}
	if body["model"] != "gpt-5.5" || body["stream"] != true || body["store"] != false {
		t.Fatalf("preview body = %#v, want model/stream/store", body)
	}
	if body["tool_choice"] != "required" {
		t.Fatalf("tool_choice = %#v, want required", body["tool_choice"])
	}
	tools, ok := body["tools"].([]map[string]string)
	if !ok || len(tools) != 1 || tools[0]["type"] != "web_search" {
		t.Fatalf("tools = %#v, want one web_search tool", body["tools"])
	}
	input, ok := body["input"].([]map[string]any)
	if !ok || len(input) != 1 {
		t.Fatalf("input = %#v, want redacted input item shape", body["input"])
	}
	if input[0]["type"] != "message" || input[0]["role"] != "user" || input[0]["content"] != "present" {
		t.Fatalf("input shape = %#v, want redacted user message", input)
	}
	reasoning, ok := body["reasoning"].(map[string]string)
	if !ok || reasoning["effort"] != "high" {
		t.Fatalf("reasoning = %#v, want effort=high preview", body["reasoning"])
	}
	for _, key := range []string{"instructions", "prompt_cache_key"} {
		if body[key] != "present" {
			t.Fatalf("body[%s] = %#v, want present", key, body[key])
		}
	}
	for _, key := range []string{"previous_response_id", "context_management", "prompt_cache_retention", "max_output_tokens", "include", "web_search_preview"} {
		if body[key] != "omitted" {
			t.Fatalf("body[%s] = %#v, want omitted", key, body[key])
		}
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Marshal(report) error = %v", err)
	}
	for _, leaked := range []string{
		"xelyon openai subscription native web search smoke",
		"user-secret",
		"pass-secret",
		"query-secret",
		"frag-secret",
	} {
		if strings.Contains(string(encoded), leaked) {
			t.Fatalf("preview leaked %q:\n%s", leaked, string(encoded))
		}
	}
}

func TestSubscriptionDiagnosticsWebSearchSmokeUsesOAuthTransport(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())
	t.Setenv("OPENAI_API_KEY", "platform-key-must-not-be-used")

	var authorization string
	var originator string
	var raw map[string]any
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		originator = r.Header.Get("originator")
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"type":"response.created","response":{"id":"resp_web_search_smoke"}}`,
			``,
			`data: {"type":"response.web_search_call.in_progress","item_id":"ws_smoke"}`,
			``,
			`data: {"type":"response.output_text.delta","delta":"Subscription web search smoke found a source."}`,
			``,
			`data: {"type":"response.output_item.done","item":{"type":"web_search_call","id":"ws_smoke","status":"completed","action":{"sources":[{"title":"Smoke source","url":"https://docs.example.test/subscription-smoke"}]}}}`,
			``,
			`data: {"type":"response.completed","response":{"id":"resp_web_search_smoke","usage":{"input_tokens":9,"output_tokens":6,"input_tokens_details":{"cached_tokens":1},"output_tokens_details":{"reasoning_tokens":2}}}}`,
			``,
			`data: [DONE]`,
		}, "\n")))
	})
	t.Setenv(subscriptionEndpointEnv, server.URL)
	if err := SaveSubscriptionCredential(DefaultSubscriptionAuthConfig(), SubscriptionCredential{
		AccessToken:  "oauth-access-token",
		RefreshToken: "oauth-refresh-token",
		AccountID:    "acct_1234abcd",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "high"

	report := DiagnoseOpenAISubscription(context.Background(), SubscriptionDiagnosticOptions{
		Config:         cfg,
		Model:          "gpt-5.5",
		CatalogModel:   "gpt-5.5",
		RunSmoke:       true,
		WebSearchSmoke: true,
	})

	if authorization != "Bearer oauth-access-token" || strings.Contains(authorization, "platform-key") {
		t.Fatalf("Authorization = %q, want OAuth bearer and no OPENAI_API_KEY", authorization)
	}
	if originator != "xelyon" {
		t.Fatalf("originator = %q, want xelyon", originator)
	}
	if raw["model"] != "gpt-5.5" || raw["stream"] != true || raw["store"] != false || raw["tool_choice"] != "required" {
		t.Fatalf("request body = %#v, want native web_search streaming store=false", raw)
	}
	input, ok := raw["input"].([]any)
	if !ok || len(input) != 1 {
		t.Fatalf("input = %#v, want one Responses input item", raw["input"])
	}
	tools, ok := raw["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v, want one web_search tool", raw["tools"])
	}
	tool, ok := tools[0].(map[string]any)
	if !ok || tool["type"] != "web_search" {
		t.Fatalf("tool = %#v, want web_search type", tools[0])
	}
	reasoning, ok := raw["reasoning"].(map[string]any)
	if !ok || reasoning["effort"] != "high" {
		t.Fatalf("reasoning = %#v, want effort=high", raw["reasoning"])
	}
	for _, forbidden := range []string{"previous_response_id", "context_management", "prompt_cache_retention", "max_output_tokens", "include", "web_search_preview"} {
		if _, ok := raw[forbidden]; ok {
			t.Fatalf("%s should be omitted: %#v", forbidden, raw)
		}
	}

	smoke := subscriptionDiagnosticTestCheck(t, report.Checks, "smoke")
	if smoke.Status != DiagnosticStatusOK {
		t.Fatalf("smoke check = %+v, want ok", smoke)
	}
	if report.Smoke == nil || !report.Smoke.WebSearchPayload || report.Smoke.WebSearchCallCount != 1 || len(report.Smoke.Requests) != 1 {
		t.Fatalf("Smoke = %#v, want one web_search smoke request", report.Smoke)
	}
	request := report.Smoke.Requests[0]
	if request.Name != "web_search" || !request.WebSearchPayload || request.WebSearchCallCount != 1 {
		t.Fatalf("smoke request = %+v, want web search call count", request)
	}
	if !request.UsageObserved || request.Usage.InputTokens != 9 || request.Usage.OutputTokens != 4 || request.Usage.CachedInputTokens != 1 || request.Usage.ThinkingTokens != 2 {
		t.Fatalf("smoke usage = %+v observed=%t, want parsed usage", request.Usage, request.UsageObserved)
	}
	if !request.Cost.PricingUnavailable {
		t.Fatalf("smoke cost = %+v, want subscription pricing unavailable", request.Cost)
	}
}

func TestSubscriptionDiagnosticsWebSearchSmokeAcceptsStreamingBodyWithoutContentType(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())

	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header()["Content-Type"] = nil
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"type":"response.created","response":{"id":"resp_web_search_smoke_no_content_type"}}`,
			``,
			`data: {"type":"response.web_search_call.in_progress","item_id":"ws_smoke_no_content_type"}`,
			``,
			`data: {"type":"response.output_text.delta","delta":"Subscription web search smoke succeeded without a Content-Type header."}`,
			``,
			`data: {"type":"response.output_item.done","item":{"type":"web_search_call","id":"ws_smoke_no_content_type","status":"completed","action":{"sources":[{"title":"Smoke source","url":"https://docs.example.test/subscription-smoke-no-content-type"}]}}}`,
			``,
			`data: {"type":"response.completed","response":{"id":"resp_web_search_smoke_no_content_type"}}`,
			``,
			`data: [DONE]`,
		}, "\n")))
	})
	t.Setenv(subscriptionEndpointEnv, server.URL)
	if err := SaveSubscriptionCredential(DefaultSubscriptionAuthConfig(), SubscriptionCredential{
		AccessToken:  "oauth-access-token",
		RefreshToken: "oauth-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}

	report := DiagnoseOpenAISubscription(context.Background(), SubscriptionDiagnosticOptions{
		Config:         config.DefaultConfig(),
		Model:          "gpt-5.5",
		CatalogModel:   "gpt-5.5",
		RunSmoke:       true,
		WebSearchSmoke: true,
	})

	smoke := subscriptionDiagnosticTestCheck(t, report.Checks, "smoke")
	if smoke.Status != DiagnosticStatusOK {
		t.Fatalf("smoke check = %+v, want ok", smoke)
	}
	if report.Smoke == nil || report.Smoke.WebSearchCallCount != 1 || len(report.Smoke.Requests) != 1 {
		t.Fatalf("Smoke = %#v, want one successful web_search smoke request", report.Smoke)
	}
	request := report.Smoke.Requests[0]
	if request.WebSearchCallCount != 1 || !strings.Contains(request.Content, "Content-Type header") ||
		!strings.Contains(request.Content, "https://docs.example.test/subscription-smoke-no-content-type") {
		t.Fatalf("smoke request = %+v, want parsed SSE body without Content-Type header", request)
	}
}

func TestSubscriptionDiagnosticsWebSearchSmokeFailsWithoutObservedCall(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())

	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"Found a source without an observed call.","annotations":[{"title":"Source","url":"https://docs.example.test/source"}]}]}}`,
			``,
			`data: {"type":"response.completed","response":{"id":"resp_web_search_no_call"}}`,
			``,
			`data: [DONE]`,
		}, "\n")))
	})
	t.Setenv(subscriptionEndpointEnv, server.URL)
	if err := SaveSubscriptionCredential(DefaultSubscriptionAuthConfig(), SubscriptionCredential{
		AccessToken:  "oauth-access-token",
		RefreshToken: "oauth-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}

	report := DiagnoseOpenAISubscription(context.Background(), SubscriptionDiagnosticOptions{
		Config:         config.DefaultConfig(),
		Model:          "gpt-5.5",
		CatalogModel:   "gpt-5.5",
		RunSmoke:       true,
		WebSearchSmoke: true,
	})

	smoke := subscriptionDiagnosticTestCheck(t, report.Checks, "smoke")
	if smoke.Status != DiagnosticStatusFail || !strings.Contains(smoke.Detail, "web_search_call") {
		t.Fatalf("smoke check = %+v, want fail mentioning web_search_call", smoke)
	}
	if report.Smoke == nil || len(report.Smoke.Requests) != 1 {
		t.Fatalf("Smoke = %#v, want one failed request", report.Smoke)
	}
	if got := report.Smoke.Requests[0].Error; !strings.Contains(got, "web_search_call") {
		t.Fatalf("smoke request error = %q, want web_search_call", got)
	}
}

func TestSubscriptionDiagnosticsWebSearchSmokeErrorWithoutParsedResultLeavesContentEmpty(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())

	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":"not an event stream"}`))
	})
	t.Setenv(subscriptionEndpointEnv, server.URL)
	if err := SaveSubscriptionCredential(DefaultSubscriptionAuthConfig(), SubscriptionCredential{
		AccessToken:  "oauth-access-token",
		RefreshToken: "oauth-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}

	report := DiagnoseOpenAISubscription(context.Background(), SubscriptionDiagnosticOptions{
		Config:         config.DefaultConfig(),
		Model:          "gpt-5.5",
		CatalogModel:   "gpt-5.5",
		RunSmoke:       true,
		WebSearchSmoke: true,
	})

	smoke := subscriptionDiagnosticTestCheck(t, report.Checks, "smoke")
	if smoke.Status != DiagnosticStatusFail || !strings.Contains(smoke.Detail, "web_search_call or source URL") {
		t.Fatalf("smoke check = %+v, want parser/validator failure", smoke)
	}
	if report.Smoke == nil || len(report.Smoke.Requests) != 1 {
		t.Fatalf("Smoke = %#v, want one failed request", report.Smoke)
	}
	request := report.Smoke.Requests[0]
	if request.Content != "" || report.Smoke.Content != "" {
		t.Fatalf("content = request %q aggregate %q, want empty on transport/parse failure", request.Content, report.Smoke.Content)
	}
	encoded, err := json.Marshal(report.Smoke)
	if err != nil {
		t.Fatalf("Marshal(Smoke) error = %v", err)
	}
	if strings.Contains(string(encoded), `"content"`) {
		t.Fatalf("failed smoke JSON should omit content: %s", encoded)
	}
}

func TestSubscriptionDiagnosticsWebSearchCapabilityIsLocalOnly(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())

	report := DiagnoseOpenAISubscription(context.Background(), SubscriptionDiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "gpt-5.5",
		CatalogModel:         "gpt-5.5",
		Capabilities:         true,
		RequiredCapabilities: []string{"web_search"},
	})

	requireNoOpenAISubscriptionDiagnosticChecks(t, report.Checks, "auth", "endpoint")
	required := subscriptionDiagnosticTestCheck(t, report.Checks, "required_capability")
	if required.Status != DiagnosticStatusOK || !strings.Contains(required.Detail, "web_search=ok") {
		t.Fatalf("required capability = %+v, want web_search ok", required)
	}
	if report.Capabilities == nil || !report.Capabilities.WebSearch || !report.Capabilities.WebSearchKnown {
		t.Fatalf("Capabilities = %#v, want known web_search", report.Capabilities)
	}
}

func requireNoOpenAISubscriptionDiagnosticChecks(t *testing.T, checks []DiagnosticCheck, names ...string) {
	t.Helper()
	for _, name := range names {
		for _, check := range checks {
			if check.Name == name {
				t.Fatalf("check %q present: %+v", name, check)
			}
		}
	}
}
