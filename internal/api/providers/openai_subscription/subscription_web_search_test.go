package openaisubscription

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
)

func TestSubscriptionWebSearchUsesOAuthTransportAndExactRequestShape(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())
	t.Setenv("OPENAI_API_KEY", "platform-key-must-not-be-used")

	var raw map[string]any
	var authorization string
	var accountID string
	var originator string
	var userAgent string
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		accountID = r.Header.Get("ChatGPT-Account-Id")
		originator = r.Header.Get("originator")
		userAgent = r.Header.Get("User-Agent")
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"type":"response.created","response":{"id":"resp_search"}}`,
			``,
			`data: {"type":"response.web_search_call.in_progress","item_id":"ws_1"}`,
			``,
			`data: {"type":"response.output_text.delta","delta":"OpenAI web search docs mention current guidance."}`,
			``,
			`data: {"type":"response.output_item.done","item":{"type":"web_search_call","id":"ws_1","status":"completed","action":{"sources":[{"title":"Docs","url":"https://docs.example.test/search"},{"title":"Docs duplicate","url":"https://docs.example.test/search"}]}}}`,
			``,
			`data: {"type":"response.completed","response":{"id":"resp_search","usage":{"input_tokens":13,"output_tokens":5,"input_tokens_details":{"cached_tokens":2}}}}`,
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

	var gotUsage api.Usage
	ctx := websearch.WithUsageCallback(context.Background(), func(usage api.Usage) {
		gotUsage = usage
	})
	result, err := WebSearchWithContext(ctx, "OpenAI web_search docs", "gpt-5.5")
	if err != nil {
		t.Fatalf("WebSearchWithContext() error = %v", err)
	}

	if authorization != "Bearer oauth-access-token" {
		t.Fatalf("Authorization = %q, want OAuth bearer token", authorization)
	}
	if strings.Contains(authorization, "platform-key") {
		t.Fatalf("Authorization used OPENAI_API_KEY: %q", authorization)
	}
	if accountID != "acct_1234abcd" {
		t.Fatalf("ChatGPT-Account-Id = %q, want account id", accountID)
	}
	if originator != "xelyon" {
		t.Fatalf("originator = %q, want xelyon", originator)
	}
	if !strings.HasPrefix(userAgent, "xelyon/") {
		t.Fatalf("User-Agent = %q, want xelyon prefix", userAgent)
	}

	if raw["model"] != "gpt-5.5" {
		t.Fatalf("model = %#v, want gpt-5.5", raw["model"])
	}
	if raw["stream"] != true {
		t.Fatalf("stream = %#v, want true", raw["stream"])
	}
	if raw["store"] != false {
		t.Fatalf("store = %#v, want false", raw["store"])
	}
	if raw["tool_choice"] != "required" {
		t.Fatalf("tool_choice = %#v, want required", raw["tool_choice"])
	}
	tools, ok := raw["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v, want one web_search tool", raw["tools"])
	}
	tool, ok := tools[0].(map[string]any)
	if !ok {
		t.Fatalf("tools[0] = %#v, want object", tools[0])
	}
	if tool["type"] != "web_search" {
		t.Fatalf("tools[0].type = %#v, want web_search", tool["type"])
	}
	if _, ok := tool["name"]; ok {
		t.Fatalf("tools[0] should not include function-tool name: %#v", tool)
	}
	for _, forbidden := range []string{"previous_response_id", "context_management", "prompt_cache_retention", "max_output_tokens", "include", "web_search_preview"} {
		if _, ok := raw[forbidden]; ok {
			t.Fatalf("%s should be omitted: %#v", forbidden, raw)
		}
	}
	if strings.TrimSpace(raw["input"].(string)) == "" || strings.TrimSpace(raw["instructions"].(string)) == "" {
		t.Fatalf("input/instructions missing: %#v", raw)
	}
	if strings.TrimSpace(raw["prompt_cache_key"].(string)) == "" {
		t.Fatalf("prompt_cache_key missing: %#v", raw)
	}

	if !strings.Contains(result, "Summary:\nOpenAI web search docs mention current guidance.") ||
		!strings.Contains(result, "Sources:\n\n1. Docs\n   URL: https://docs.example.test/search") ||
		strings.Contains(result, "Docs duplicate") {
		t.Fatalf("result = %q, want Summary/Sources with deduped source", result)
	}
	if gotUsage.InputTokens != 13 || gotUsage.OutputTokens != 5 || gotUsage.CachedInputTokens != 2 {
		t.Fatalf("usage = %+v, want parsed token usage", gotUsage)
	}
	if gotUsage.StorageCost != 0 || gotUsage.WebSearchCalls != 0 {
		t.Fatalf("usage = %+v, want no Platform API cost or Kimi-style web search fee", gotUsage)
	}
}

func TestSubscriptionWebSearchRejectsPlatformEndpointBeforeAuth(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())
	t.Setenv(subscriptionEndpointEnv, openAIPlatformResponsesURL)

	_, err := WebSearchWithContext(context.Background(), "query", "gpt-5.5")
	if err == nil || !strings.Contains(err.Error(), "must not use OpenAI Platform Responses API endpoint") {
		t.Fatalf("WebSearchWithContext() error = %v, want Platform endpoint rejection", err)
	}
	if strings.Contains(err.Error(), "not logged in") {
		t.Fatalf("WebSearchWithContext() error = %v, want endpoint validation before auth", err)
	}
}

func TestSubscriptionWebSearchRejectsUnsupportedModelBeforeAuth(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())

	_, err := WebSearchWithContext(context.Background(), "query", "gpt-5.2")
	if err == nil || !strings.Contains(err.Error(), "model gpt-5.2 is not supported by openai_subscription") {
		t.Fatalf("WebSearchWithContext() error = %v, want unsupported model", err)
	}
	if strings.Contains(err.Error(), "not logged in") {
		t.Fatalf("WebSearchWithContext() error = %v, want model validation before auth", err)
	}
}

func TestSubscriptionWebSearchRegistrationIsCanonicalOnly(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())
	_, err := websearch.SearchWithContext(context.Background(), subscriptionProviderKey, "query", "gpt-5.2")
	if err == nil || !strings.Contains(err.Error(), "model gpt-5.2 is not supported by openai_subscription") {
		t.Fatalf("SearchWithContext(openai_subscription) error = %v, want registered subscription adapter", err)
	}

	_, err = websearch.SearchWithContext(context.Background(), "chatgpt", "query", "gpt-5.5")
	if err == nil || !strings.Contains(err.Error(), `native web search is not registered for provider "chatgpt"`) {
		t.Fatalf("SearchWithContext(chatgpt) error = %v, want alias unregistered", err)
	}
}

func TestSubscriptionWebSearchParserAcceptsStructuredSourcesWithoutCallEvent(t *testing.T) {
	result := parseSubscriptionWebSearchStreamForTest(t, strings.Join([]string{
		`data: {"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"Found docs.","annotations":[{"title":"Docs","url":"https://docs.example.test/structured"}]}]}}`,
		``,
		`data: {"type":"response.completed","response":{"id":"resp_structured","usage":{"input_tokens":3,"output_tokens":2}}}`,
	}, "\n"))

	if err := validateSubscriptionWebSearchRuntimeResult(result); err != nil {
		t.Fatalf("validate runtime result error = %v, want nil", err)
	}
	if result.WebSearchCallCount != 0 {
		t.Fatalf("WebSearchCallCount = %d, want 0 when event naming drifted", result.WebSearchCallCount)
	}
	if result.Summary != "Found docs." || len(result.Sources) != 1 || result.Sources[0].URL != "https://docs.example.test/structured" {
		t.Fatalf("result = %#v, want message summary and annotation source", result)
	}
}

func TestSubscriptionWebSearchParserAcceptsSummaryWithExtractableURL(t *testing.T) {
	result := parseSubscriptionWebSearchStreamForTest(t, strings.Join([]string{
		`data: {"type":"response.output_text.delta","delta":"See https://docs.example.test/from-summary for details."}`,
		``,
		`data: {"type":"response.completed","response":{"id":"resp_url"}}`,
	}, "\n"))

	if err := validateSubscriptionWebSearchRuntimeResult(result); err != nil {
		t.Fatalf("validate runtime result error = %v, want nil", err)
	}
	if result.WebSearchCallCount != 0 {
		t.Fatalf("WebSearchCallCount = %d, want 0", result.WebSearchCallCount)
	}
	if len(result.Sources) != 1 || result.Sources[0].URL != "https://docs.example.test/from-summary" {
		t.Fatalf("Sources = %#v, want URL extracted from summary", result.Sources)
	}
}

func TestSubscriptionWebSearchParserSummaryOnlyFailsRuntimeAndSmoke(t *testing.T) {
	result := parseSubscriptionWebSearchStreamForTest(t, strings.Join([]string{
		`data: {"type":"response.output_text.delta","delta":"Ungrounded summary without source."}`,
		``,
		`data: {"type":"response.completed","response":{"id":"resp_summary_only"}}`,
	}, "\n"))

	if err := validateSubscriptionWebSearchRuntimeResult(result); err == nil || !strings.Contains(err.Error(), "web_search_call or source URL") {
		t.Fatalf("validate runtime result error = %v, want ungrounded failure", err)
	}
	if err := validateSubscriptionWebSearchSmokeResult(result); err == nil || !strings.Contains(err.Error(), "web_search_call") {
		t.Fatalf("validate smoke result error = %v, want missing web_search_call failure", err)
	}
}

func TestSubscriptionWebSearchParserIgnoresUnknownFutureEvents(t *testing.T) {
	result := parseSubscriptionWebSearchStreamForTest(t, strings.Join([]string{
		`data: {"type":"response.future_event","payload":{"opaque":true}}`,
		``,
		`data: {"type":"response.web_search_call.completed","item_id":"ws_future"}`,
		``,
		`data: {"type":"response.output_text.delta","delta":"Future event did not break parsing."}`,
		``,
		`data: {"type":"response.completed","response":{"id":"resp_future"}}`,
	}, "\n"))

	if err := validateSubscriptionWebSearchRuntimeResult(result); err != nil {
		t.Fatalf("validate runtime result error = %v, want nil", err)
	}
	if result.WebSearchCallCount != 1 || result.Summary != "Future event did not break parsing." {
		t.Fatalf("result = %#v, want one web search call and summary", result)
	}
}

func parseSubscriptionWebSearchStreamForTest(t *testing.T, body string) subscriptionWebSearchResult {
	t.Helper()
	resp := &http.Response{
		Header: http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:   io.NopCloser(strings.NewReader(body)),
	}
	result, err := parseSubscriptionWebSearchStream(context.Background(), resp)
	if err != nil {
		t.Fatalf("parseSubscriptionWebSearchStream() error = %v", err)
	}
	return result
}
