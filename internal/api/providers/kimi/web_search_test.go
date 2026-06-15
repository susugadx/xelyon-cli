package kimi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestWebSearchWithContext_BuiltinToolLoopPayloadAndUsage(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "test-key")
	const toolArguments = `{"query":"Moonshot AI","usage":{"total_tokens":4}}`

	var captured []map[string]any
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		captured = append(captured, body)

		switch len(captured) {
		case 1:
			kimiStreamingHandler([]string{
				`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_web","type":"builtin_function","function":{"name":"$web_search","arguments":"{\"query\":\"Moonshot AI\",\"usage\":{\"total_tokens\":4}}"}}]}}]}`,
				`{"choices":[{"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":11,"completion_tokens":2,"cached_tokens":3}}`,
			})(w, r)
		case 2:
			kimiStreamingHandler([]string{
				`{"choices":[{"delta":{"content":"Summary: Kimi search result."}}]}`,
				`{"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":21,"completion_tokens":6,"cached_tokens":5}}`,
			})(w, r)
		default:
			t.Fatalf("unexpected request count %d", len(captured))
		}
	})
	t.Setenv(kimiAPIURLEnv, server.URL)

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("kimi", config.ProviderModelConfig{
		DefaultModel:    "kimi-search-test",
		MaxOutputTokens: 123,
	})
	ctx := config.WithContext(context.Background(), cfg)
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	ctx = api.WithPromptCacheScope(ctx, api.PromptCacheScope{SessionID: "web-search-session"})
	var usages []api.Usage
	ctx = websearch.WithUsageCallback(ctx, func(usage api.Usage) {
		usages = append(usages, usage)
	})

	got, err := WebSearchWithContext(ctx, "Moonshot AI Context Caching", "")
	if err != nil {
		t.Fatalf("WebSearchWithContext() error = %v", err)
	}
	if got != "Summary: Kimi search result." {
		t.Fatalf("WebSearchWithContext() = %q, want final content", got)
	}

	if len(captured) != 2 {
		t.Fatalf("captured request count = %d, want 2", len(captured))
	}
	assertKimiWebSearchBasePayload(t, captured[0])
	assertKimiWebSearchBasePayload(t, captured[1])
	assertKimiWebSearchLoopMessages(t, captured[1]["messages"], toolArguments)
	if _, ok := captured[0]["tool_choice"]; ok {
		t.Fatalf("tool_choice = %#v, want omitted for %s", captured[0]["tool_choice"], kimiWebSearchToolName)
	}
	if _, ok := captured[0]["max_tokens"]; ok {
		t.Fatal("max_tokens should be omitted")
	}
	if len(usages) != 3 {
		t.Fatalf("usage callback count = %d, want 3", len(usages))
	}
	if usages[0].InputTokens != 11 || usages[0].OutputTokens != 2 || usages[0].CachedInputTokens != 3 {
		t.Fatalf("first usage = %+v, want input=11 output=2 cached=3", usages[0])
	}
	if usages[1].WebSearchCalls != 1 || usages[1].WebSearchResultTokens != 4 || usages[1].StorageCost != kimiWebSearchCallFeeUSD {
		t.Fatalf("web search usage = %+v, want one call, 4 observed result tokens, fee %.4f", usages[1], kimiWebSearchCallFeeUSD)
	}
	if usages[1].InputTokens != 0 || usages[1].OutputTokens != 0 {
		t.Fatalf("web search usage tokens = input %d output %d, want no token double count", usages[1].InputTokens, usages[1].OutputTokens)
	}
	if usages[2].InputTokens != 21 || usages[2].OutputTokens != 6 || usages[2].CachedInputTokens != 5 {
		t.Fatalf("second token usage = %+v, want input=21 output=6 cached=5", usages[2])
	}
}

func TestKimiWebSearchToolCallUsage_ParsesResultTokenObservations(t *testing.T) {
	usage := kimiWebSearchToolCallUsage([]api.OpenAIToolCall{
		{Function: api.OpenAIToolCallFunction{Name: kimiWebSearchToolName, Arguments: `{"usage":{"total_tokens":12}}`}},
		{Function: api.OpenAIToolCallFunction{Name: kimiWebSearchToolName, Arguments: `{"total_tokens":7}`}},
		{Function: api.OpenAIToolCallFunction{Name: kimiWebSearchToolName, Arguments: `not-json`}},
		{Function: api.OpenAIToolCallFunction{Name: "other", Arguments: `{"total_tokens":99}`}},
	})

	if usage == nil {
		t.Fatal("kimiWebSearchToolCallUsage() = nil, want usage")
	}
	if usage.WebSearchCalls != 3 {
		t.Fatalf("WebSearchCalls = %d, want 3", usage.WebSearchCalls)
	}
	if usage.WebSearchResultTokens != 19 {
		t.Fatalf("WebSearchResultTokens = %d, want 19", usage.WebSearchResultTokens)
	}
	if usage.StorageCost != 3*kimiWebSearchCallFeeUSD {
		t.Fatalf("StorageCost = %f, want %f", usage.StorageCost, 3*kimiWebSearchCallFeeUSD)
	}
	if usage.InputTokens != 0 || usage.OutputTokens != 0 {
		t.Fatalf("token usage = input %d output %d, want no token double count", usage.InputTokens, usage.OutputTokens)
	}
}

func TestWebSearchWithContext_InvalidToolArgumentsStillReplayAndReportCallFee(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "test-key")
	var captured []map[string]any
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		captured = append(captured, body)
		switch len(captured) {
		case 1:
			kimiStreamingHandler([]string{
				`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_web","type":"builtin_function","function":{"name":"$web_search","arguments":"not-json"}}]}}]}`,
				`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
			})(w, r)
		case 2:
			kimiStreamingHandler([]string{
				`{"choices":[{"delta":{"content":"ok after invalid arguments replay"}}]}`,
				`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
			})(w, r)
		default:
			t.Fatalf("unexpected request count %d", len(captured))
		}
	})
	t.Setenv(kimiAPIURLEnv, server.URL)

	ctx, _, _ := newKimiTestContext(t, false)
	ctx = websearch.WithUsageCallback(ctx, func(usage api.Usage) {
		if usage.WebSearchCalls != 1 || usage.StorageCost != kimiWebSearchCallFeeUSD {
			t.Fatalf("web search usage = %+v, want one fee observation", usage)
		}
	})

	got, err := WebSearchWithContext(ctx, "invalid arguments", "kimi-k2.6")
	if err != nil {
		t.Fatalf("WebSearchWithContext() error = %v", err)
	}
	if got != "ok after invalid arguments replay" {
		t.Fatalf("WebSearchWithContext() = %q, want final content", got)
	}
	if len(captured) != 2 {
		t.Fatalf("captured request count = %d, want 2", len(captured))
	}
	assertKimiWebSearchLoopMessages(t, captured[1]["messages"], "not-json")
}

func TestWebSearchWithContext_MoonshotAliasUsesAliasDefaultModel(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "test-key")
	var captured map[string]any
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		kimiStreamingHandler([]string{
			`{"choices":[{"delta":{"content":"alias result"}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2}}`,
		})(w, r)
	})
	t.Setenv(kimiAPIURLEnv, server.URL)

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("moonshot", config.ProviderModelConfig{
		DefaultModel:    "moonshot-search-default",
		MaxOutputTokens: 77,
	})
	ctx := config.WithContext(context.Background(), cfg)
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

	got, err := websearch.SearchWithContext(ctx, "moonshot", "alias query", "")
	if err != nil {
		t.Fatalf("SearchWithContext(moonshot) error = %v", err)
	}
	if got != "alias result" {
		t.Fatalf("SearchWithContext(moonshot) = %q, want alias result", got)
	}
	if captured["model"] != "moonshot-search-default" {
		t.Fatalf("model = %#v, want moonshot alias default", captured["model"])
	}
	if captured["max_completion_tokens"] != float64(77) {
		t.Fatalf("max_completion_tokens = %#v, want 77", captured["max_completion_tokens"])
	}
}

func TestWebSearchWithContext_ForcedThinkingModelOmitsDisabledThinking(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "test-key")
	var captured map[string]any
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		kimiStreamingHandler([]string{
			`{"choices":[{"delta":{"content":"forced thinking model result"}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2}}`,
		})(w, r)
	})
	t.Setenv(kimiAPIURLEnv, server.URL)

	ctx, _, _ := newKimiTestContext(t, false)
	got, err := WebSearchWithContext(ctx, "forced thinking", "kimi-k2-thinking")
	if err != nil {
		t.Fatalf("WebSearchWithContext() error = %v", err)
	}
	if got != "forced thinking model result" {
		t.Fatalf("WebSearchWithContext() = %q, want forced thinking model result", got)
	}
	if captured["model"] != "kimi-k2-thinking" {
		t.Fatalf("model = %#v, want kimi-k2-thinking", captured["model"])
	}
	if _, ok := captured["thinking"]; ok {
		t.Fatalf("thinking = %#v, want omitted for forced thinking model", captured["thinking"])
	}
}

func TestBuildKimiWebSearchRequest_OmitsDisabledThinkingForForcedCatalogModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("moonshot", config.ProviderModelConfig{
		DefaultModel: "corp-kimi",
		CatalogModel: "kimi-k2-thinking",
	})
	ctx := config.WithContext(context.Background(), cfg)
	req := buildKimiWebSearchRequest(ctx, initialKimiWebSearchMessages("catalog forced"), "corp-kimi", "moonshot")
	if req.Model != "corp-kimi" {
		t.Fatalf("model = %q, want corp-kimi", req.Model)
	}
	if req.Thinking != nil {
		t.Fatalf("thinking = %#v, want omitted when catalog model is forced-thinking", req.Thinking)
	}
}

func TestBuildKimiWebSearchRequest_K27FallsBackToK26(t *testing.T) {
	ctx, _, _ := newKimiTestContext(t, false)
	req := buildKimiWebSearchRequest(ctx, initialKimiWebSearchMessages("k2.7 search"), "kimi-k2.7-code", "kimi")
	if req.Model != "kimi-k2.6" {
		t.Fatalf("model = %q, want kimi-k2.6 fallback", req.Model)
	}
	if req.Thinking == nil || req.Thinking["type"] != "disabled" {
		t.Fatalf("thinking = %#v, want disabled for Kimi built-in web_search fallback request", req.Thinking)
	}
}

func TestBuildKimiWebSearchRequest_K27FallbackUsesK26PromptCacheKey(t *testing.T) {
	ctx, _, _ := newKimiTestContext(t, false)
	ctx = api.WithPromptCacheScope(ctx, api.PromptCacheScope{SessionID: "k2.7-web-search-cache-session"})

	req := buildKimiWebSearchRequest(ctx, initialKimiWebSearchMessages("k2.7 cache key"), "kimi-k2.7-code", "kimi")
	wantKey := buildKimiPromptCacheKey(ctx, "kimi-k2.6", kimiWebSearchSystemPrompt)
	k27Key := buildKimiPromptCacheKey(ctx, "kimi-k2.7-code", kimiWebSearchSystemPrompt)

	if req.Model != "kimi-k2.6" {
		t.Fatalf("model = %q, want kimi-k2.6 fallback", req.Model)
	}
	if req.PromptCacheKey != wantKey {
		t.Fatalf("prompt_cache_key = %q, want K2.6 key %q", req.PromptCacheKey, wantKey)
	}
	if req.PromptCacheKey == k27Key {
		t.Fatalf("prompt_cache_key = %q, must not reuse K2.7 key for K2.6 fallback request", req.PromptCacheKey)
	}
}

func TestBuildKimiWebSearchRequest_K27CatalogAliasFallbackUsesK26PromptCacheKey(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("moonshot", config.ProviderModelConfig{
		DefaultModel: "corp-kimi-code",
		CatalogModel: "kimi-k2.7-code",
	})
	ctx := config.WithContext(context.Background(), cfg)
	ctx = api.WithPromptCacheScope(ctx, api.PromptCacheScope{SessionID: "k2.7-catalog-web-search-cache-session"})

	req := buildKimiWebSearchRequest(ctx, initialKimiWebSearchMessages("k2.7 catalog cache key"), "corp-kimi-code", "moonshot")
	wantKey := buildKimiPromptCacheKey(ctx, "kimi-k2.6", kimiWebSearchSystemPrompt)
	aliasKey := buildKimiPromptCacheKey(ctx, "corp-kimi-code", kimiWebSearchSystemPrompt)

	if req.Model != "kimi-k2.6" {
		t.Fatalf("model = %q, want kimi-k2.6 fallback", req.Model)
	}
	if req.PromptCacheKey != wantKey {
		t.Fatalf("prompt_cache_key = %q, want K2.6 key %q", req.PromptCacheKey, wantKey)
	}
	if req.PromptCacheKey == aliasKey {
		t.Fatalf("prompt_cache_key = %q, must not reuse alias key for K2.6 fallback request", req.PromptCacheKey)
	}
}

func TestBuildKimiWebSearchRequest_K27CatalogFallsBackToK26(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("moonshot", config.ProviderModelConfig{
		DefaultModel: "corp-kimi-code",
		CatalogModel: "kimi-k2.7-code",
	})
	ctx := config.WithContext(context.Background(), cfg)
	req := buildKimiWebSearchRequest(ctx, initialKimiWebSearchMessages("catalog k2.7 search"), "corp-kimi-code", "moonshot")
	if req.Model != "kimi-k2.6" {
		t.Fatalf("model = %q, want kimi-k2.6 fallback for kimi-k2.7-code catalog model", req.Model)
	}
	if req.Thinking == nil || req.Thinking["type"] != "disabled" {
		t.Fatalf("thinking = %#v, want disabled for Kimi built-in web_search fallback request", req.Thinking)
	}
}

func TestWebSearchWithContext_ErrorsAfterMaxToolLoops(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "test-key")
	requests := 0
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		kimiStreamingHandler([]string{
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_web","type":"builtin_function","function":{"name":"$web_search","arguments":"{\"query\":\"loop\"}"}}]}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
		})(w, r)
	})
	t.Setenv(kimiAPIURLEnv, server.URL)

	ctx, _, _ := newKimiTestContext(t, false)
	var webSearchCalls int
	var fee float64
	ctx = websearch.WithUsageCallback(ctx, func(usage api.Usage) {
		webSearchCalls += usage.WebSearchCalls
		fee += usage.StorageCost
	})
	_, err := WebSearchWithContext(ctx, "loop", "kimi-k2.6")
	if err == nil {
		t.Fatal("WebSearchWithContext() error = nil, want max loop error")
	}
	if !strings.Contains(err.Error(), "did not complete within 3 requests") {
		t.Fatalf("error = %v, want max request message", err)
	}
	if requests != kimiWebSearchMaxRequests {
		t.Fatalf("requests = %d, want %d", requests, kimiWebSearchMaxRequests)
	}
	if webSearchCalls != kimiWebSearchMaxRequests {
		t.Fatalf("webSearchCalls = %d, want %d charged tool call observations", webSearchCalls, kimiWebSearchMaxRequests)
	}
	if fee != float64(kimiWebSearchMaxRequests)*kimiWebSearchCallFeeUSD {
		t.Fatalf("fee = %f, want %f", fee, float64(kimiWebSearchMaxRequests)*kimiWebSearchCallFeeUSD)
	}
}

func TestWebSearchWithContext_ErrorsOnIncompleteFinishReasonWithContent(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "test-key")
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		kimiStreamingHandler([]string{
			`{"choices":[{"delta":{"content":"partial Kimi search result"}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"length"}],"usage":{"prompt_tokens":5,"completion_tokens":2}}`,
		})(w, r)
	})
	t.Setenv(kimiAPIURLEnv, server.URL)

	ctx, _, _ := newKimiTestContext(t, false)
	got, err := WebSearchWithContext(ctx, "partial", "kimi-k2.6")
	if err == nil {
		t.Fatal("WebSearchWithContext() error = nil, want incomplete finish_reason error")
	}
	if got != "" {
		t.Fatalf("WebSearchWithContext() = %q, want empty result on incomplete finish_reason", got)
	}
	if !strings.Contains(err.Error(), `finish_reason "length"`) {
		t.Fatalf("error = %v, want finish_reason length", err)
	}
}

func assertKimiWebSearchBasePayload(t *testing.T, body map[string]any) {
	t.Helper()
	if body["model"] != "kimi-search-test" {
		t.Fatalf("model = %#v, want kimi-search-test", body["model"])
	}
	if body["max_completion_tokens"] != float64(123) {
		t.Fatalf("max_completion_tokens = %#v, want 123", body["max_completion_tokens"])
	}
	if body["stream"] != true {
		t.Fatalf("stream = %#v, want true", body["stream"])
	}
	streamOptions, ok := body["stream_options"].(map[string]any)
	if !ok || streamOptions["include_usage"] != true {
		t.Fatalf("stream_options = %#v, want include_usage=true", body["stream_options"])
	}
	if key, ok := body["prompt_cache_key"].(string); !ok || key == "" || !strings.HasPrefix(key, "xelyon:kimi:v1:") {
		t.Fatalf("prompt_cache_key = %#v, want session-aware Kimi key", body["prompt_cache_key"])
	}
	thinking, ok := body["thinking"].(map[string]any)
	if !ok || thinking["type"] != "disabled" {
		t.Fatalf("thinking = %#v, want disabled", body["thinking"])
	}
	toolsPayload, ok := body["tools"].([]any)
	if !ok || len(toolsPayload) != 1 {
		t.Fatalf("tools = %#v, want one builtin tool", body["tools"])
	}
	tool, ok := toolsPayload[0].(map[string]any)
	if !ok || tool["type"] != "builtin_function" {
		t.Fatalf("tool = %#v, want builtin_function", toolsPayload[0])
	}
	function, ok := tool["function"].(map[string]any)
	if !ok || function["name"] != kimiWebSearchToolName {
		t.Fatalf("tool.function = %#v, want %s", tool["function"], kimiWebSearchToolName)
	}
}

func assertKimiWebSearchLoopMessages(t *testing.T, rawMessages any, wantToolContent string) {
	t.Helper()
	messages, ok := rawMessages.([]any)
	if !ok || len(messages) != 4 {
		t.Fatalf("messages = %#v, want system + user + assistant + tool", rawMessages)
	}
	assistant, ok := messages[2].(map[string]any)
	if !ok || assistant["role"] != "assistant" {
		t.Fatalf("assistant message = %#v, want role assistant", messages[2])
	}
	toolCallsPayload, ok := assistant["tool_calls"].([]any)
	if !ok || len(toolCallsPayload) != 1 {
		t.Fatalf("assistant tool_calls = %#v, want one tool call", assistant["tool_calls"])
	}
	toolCall, ok := toolCallsPayload[0].(map[string]any)
	if !ok || toolCall["id"] != "call_web" || toolCall["type"] != "builtin_function" {
		t.Fatalf("assistant tool_call = %#v, want returned builtin_function call", toolCallsPayload[0])
	}
	function, ok := toolCall["function"].(map[string]any)
	if !ok || function["name"] != kimiWebSearchToolName || function["arguments"] != wantToolContent {
		t.Fatalf("assistant tool_call.function = %#v, want %s with exact arguments", toolCall["function"], kimiWebSearchToolName)
	}
	toolMessage, ok := messages[3].(map[string]any)
	if !ok || toolMessage["role"] != "tool" || toolMessage["tool_call_id"] != "call_web" || toolMessage["name"] != kimiWebSearchToolName {
		t.Fatalf("tool message = %#v, want role/tool_call_id/name", messages[3])
	}
	if toolMessage["content"] != wantToolContent {
		t.Fatalf("tool message content = %#v, want exact arguments %s", toolMessage["content"], wantToolContent)
	}
	if _, ok := toolMessage["tool_name"]; ok {
		t.Fatalf("tool message = %#v, want Kimi name field not tool_name", toolMessage)
	}
}
