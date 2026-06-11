package openaisubscription

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

var _ api.CompactCapable = (*SubscriptionProvider)(nil)

func TestSubscriptionSupportedModelsExactAllowlist(t *testing.T) {
	want := []string{"gpt-5.5", "gpt-5.4", "gpt-5.4-mini", "gpt-5.3-codex-spark"}
	got := SubscriptionSupportedModels()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("SubscriptionSupportedModels() = %v, want %v", got, want)
	}
	got[0] = "mutated"
	if again := SubscriptionSupportedModels(); again[0] != "gpt-5.5" {
		t.Fatalf("SubscriptionSupportedModels() did not return a clone: %v", again)
	}
}

func TestSubscriptionProviderSupportsCompactDefaultAndDisabledEndpoint(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir()+"/auth")
	if !NewSubscription().SupportsCompact() {
		t.Fatal("SupportsCompact() = false, want true for default subscription compact endpoint")
	}

	t.Setenv(subscriptionCompactEndpointEnv, "")
	if NewSubscription().SupportsCompact() {
		t.Fatal("SupportsCompact() = true, want false when compact endpoint is explicitly disabled")
	}
}

func TestValidateSubscriptionModel(t *testing.T) {
	for _, model := range []string{"gpt-5.5", "gpt-5.4", "gpt-5.4-mini", "gpt-5.3-codex-spark"} {
		t.Run("allowed/"+model, func(t *testing.T) {
			if err := ValidateSubscriptionModel(model); err != nil {
				t.Fatalf("ValidateSubscriptionModel(%q) error = %v, want nil", model, err)
			}
		})
	}

	for _, model := range []string{"gpt-5.3-codex", "gpt-5.2", "gpt-4.1"} {
		t.Run("rejected/"+model, func(t *testing.T) {
			err := ValidateSubscriptionModel(model)
			if err == nil {
				t.Fatalf("ValidateSubscriptionModel(%q) error = nil, want unsupported", model)
			}
			text := err.Error()
			if !strings.Contains(text, "model "+model+" is not supported by openai_subscription.") ||
				!strings.Contains(text, "Supported models: gpt-5.5, gpt-5.4, gpt-5.4-mini, gpt-5.3-codex-spark.") ||
				!strings.Contains(text, "Use provider openai if you need OpenAI Platform API / legacy models") {
				t.Fatalf("unsupported error = %q", text)
			}
		})
	}
}

func TestSubscriptionProviderLoginRequiredAfterModelValidation(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir()+"/auth")

	provider := NewSubscription()
	if provider.Name() != "OpenAI Subscription" {
		t.Fatalf("Name() = %q, want OpenAI Subscription", provider.Name())
	}
	if provider.SupportsImages() {
		t.Fatal("SupportsImages() = true, want false")
	}
	if !provider.IsFunctionCallingEnabled() {
		t.Fatal("IsFunctionCallingEnabled() = false, want true")
	}
	if got := provider.RuntimeProviderName(); got != "openai_subscription" {
		t.Fatalf("RuntimeProviderName() = %q, want openai_subscription", got)
	}
	if got := provider.ProviderConfigKey(); got != "openai_subscription" {
		t.Fatalf("ProviderConfigKey() = %q, want openai_subscription", got)
	}

	_, err := provider.ChatWithTools(context.Background(), "", nil, "gpt-5.5")
	if err == nil {
		t.Fatal("ChatWithTools() error = nil, want login required")
	}
	if !strings.Contains(err.Error(), "openai_subscription is not logged in.") ||
		!strings.Contains(err.Error(), "Run: xelyon auth openai-subscription login") {
		t.Fatalf("ChatWithTools() error = %q, want login suggestion", err.Error())
	}
}

func TestSubscriptionProviderRejectsUnsupportedModelBeforeLogin(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir()+"/auth")

	provider := NewSubscription()
	_, err := provider.ChatWithTools(context.Background(), "", nil, "gpt-5.2")
	if err == nil {
		t.Fatal("ChatWithTools(gpt-5.2) error = nil, want unsupported model")
	}
	if strings.Contains(err.Error(), "not logged in") {
		t.Fatalf("ChatWithTools(gpt-5.2) error = %q, want model error before login error", err.Error())
	}
}

func TestSubscriptionProviderRejectsInvalidResponsesEndpointBeforeAuth(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir()+"/auth")
	t.Setenv(subscriptionEndpointEnv, "ftp://chatgpt.example.test/backend-api/codex/responses")

	_, err := NewSubscription().ChatWithTools(context.Background(), "", []api.Message{{Role: "user", Content: "hello"}}, "gpt-5.5")
	if err == nil || !strings.Contains(err.Error(), "subscription endpoint must use http or https") {
		t.Fatalf("ChatWithTools() error = %v, want endpoint scheme validation", err)
	}
	if strings.Contains(err.Error(), "not logged in") {
		t.Fatalf("ChatWithTools() error = %v, want endpoint validation before auth", err)
	}
}

func TestSubscriptionProviderRefusesOpenAIPlatformResponsesEndpointBeforeAuth(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir()+"/auth")
	t.Setenv(subscriptionEndpointEnv, openAIPlatformResponsesURL)

	_, err := NewSubscription().ChatWithTools(context.Background(), "", []api.Message{{Role: "user", Content: "hello"}}, "gpt-5.5")
	if err == nil || !strings.Contains(err.Error(), "must not use OpenAI Platform Responses API endpoint") {
		t.Fatalf("ChatWithTools() error = %v, want forbidden Platform endpoint", err)
	}
	if strings.Contains(err.Error(), "not logged in") {
		t.Fatalf("ChatWithTools() error = %v, want endpoint validation before auth", err)
	}
}

func TestSubscriptionProviderRejectsNonXelyonOriginatorBeforeAuth(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir()+"/auth")
	t.Setenv(subscriptionOriginatorEnv, "opencode")

	_, err := NewSubscription().ChatWithTools(context.Background(), "", []api.Message{{Role: "user", Content: "hello"}}, "gpt-5.5")
	if err == nil || !strings.Contains(err.Error(), "subscription originator must be xelyon") {
		t.Fatalf("ChatWithTools() error = %v, want originator validation", err)
	}
	if strings.Contains(err.Error(), "not logged in") {
		t.Fatalf("ChatWithTools() error = %v, want originator validation before auth", err)
	}
}

func TestSubscriptionProviderResponsesRequestUsesOAuthTransportAndV2Shape(t *testing.T) {
	authDir := t.TempDir()
	t.Setenv(subscriptionAuthDirEnv, authDir)
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
			`data: {"type":"response.created","response":{"id":"resp_sub"}}`,
			``,
			`data: {"type":"response.output_text.delta","delta":"hello subscription"}`,
			``,
			`data: {"type":"response.completed","response":{"usage":{"input_tokens":10,"output_tokens":4,"input_tokens_details":{"cached_tokens":2},"output_tokens_details":{"reasoning_tokens":1}}}}`,
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
	provider := NewSubscription()
	provider.SetMCPTools([]api.ToolDefinition{{
		Name:        "dummy_tool",
		Description: "dummy tool",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"value": map[string]interface{}{"type": "string"},
			},
		},
	}})
	provider.SetUsageCallback(func(usage api.Usage) {
		gotUsage = usage
	})
	content, err := provider.ChatWithTools(newOpenAITestContext(t, false), "system prompt", []api.Message{{Role: "user", Content: "hello"}}, "gpt-5.5")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if !strings.Contains(content, "hello subscription") {
		t.Fatalf("content = %q, want subscription text", content)
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
	if raw["instructions"] != "system prompt" {
		t.Fatalf("instructions = %#v, want system prompt", raw["instructions"])
	}
	if _, ok := raw["previous_response_id"]; ok {
		t.Fatalf("previous_response_id should be omitted: %#v", raw)
	}
	if _, ok := raw["context_management"]; ok {
		t.Fatalf("context_management should be omitted: %#v", raw)
	}
	if raw["prompt_cache_key"] == "" {
		t.Fatalf("prompt_cache_key missing: %#v", raw)
	}
	if _, ok := raw["prompt_cache_retention"]; ok {
		t.Fatalf("prompt_cache_retention should be omitted by subscription policy: %#v", raw)
	}
	if _, ok := raw["max_output_tokens"]; ok {
		t.Fatalf("max_output_tokens should be omitted by default: %#v", raw)
	}
	tools, ok := raw["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("tools = %#v, want tool payload", raw["tools"])
	}
	if gotUsage.InputTokens != 10 || gotUsage.CachedInputTokens != 2 || gotUsage.ThinkingTokens != 1 {
		t.Fatalf("usage = %+v, want parsed subscription usage", gotUsage)
	}
	replayItems := provider.LastOpenAIResponsesInputItems()
	if len(replayItems) != 1 ||
		replayItems[0].Type != "message" ||
		replayItems[0].Role != "assistant" ||
		replayItems[0].Content != "hello subscription" {
		t.Fatalf("LastOpenAIResponsesInputItems() = %#v, want assistant replay text", replayItems)
	}
}

func TestSubscriptionProviderDebugOutputUsesStructuralPreview(t *testing.T) {
	authDir := t.TempDir()
	t.Setenv(subscriptionAuthDirEnv, authDir)
	t.Setenv("XELYON_DEBUG_OPENAI_SUBSCRIPTION", "1")

	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"type":"response.created","response":{"id":"resp_sub_debug"}}`,
			``,
			`data: {"type":"response.output_item.added","output_index":0,"item":{"type":"reasoning","id":"rs_debug","encrypted_content":"stream-encrypted-secret"}}`,
			``,
			`data: {"type":"response.output_text.delta","output_index":1,"delta":"stream-text-secret"}`,
			``,
			`data: {"type":"response.completed","response":{"usage":{"input_tokens":11,"output_tokens":3}}}`,
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

	var debugOut bytes.Buffer
	runtime := ui.NewRuntime(strings.NewReader(""), &bytes.Buffer{}, &debugOut)
	ctx := ui.WithRuntime(context.Background(), runtime)
	ctx = config.WithContext(ctx, config.DefaultConfig())
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

	assistant := api.Message{Role: "assistant", Content: "assistant-content-secret"}
	assistant.SetOpenAIResponsesInputItems([]api.InputItem{
		{Type: "reasoning", ID: "rs_1", EncryptedContent: "request-encrypted-secret"},
		{Type: "function_call", CallID: "call_1", Name: "debug_tool", Arguments: `{"value":"request-argument-secret"}`},
	})
	toolResult := api.Message{Role: "tool", ToolCallID: "call_1", ToolName: "debug_tool", Content: "tool-output-secret"}
	toolResult.SetOpenAIResponsesInputItems([]api.InputItem{
		{Type: "function_call_output", CallID: "call_1", Output: "tool-output-secret"},
	})

	provider := NewSubscription()
	provider.SetMCPTools([]api.ToolDefinition{{
		Name:        "debug_tool",
		Description: "tool-description-secret",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"value": map[string]interface{}{"type": "string"},
			},
		},
	}})

	content, err := provider.ChatWithTools(ctx, "system-prompt-secret", []api.Message{
		{Role: "user", Content: "user-prompt-secret"},
		assistant,
		toolResult,
		{Role: "user", Content: "followup-user-secret"},
	}, "gpt-5.5")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if !strings.Contains(content, "stream-text-secret") {
		t.Fatalf("content = %q, want streamed text", content)
	}

	debug := debugOut.String()
	for _, want := range []string{
		"[DEBUG OpenAI Subscription Responses] Request preview:",
		`"instructions": "present"`,
		`"content": "present"`,
		`"encrypted": "present"`,
		`"arguments": "present"`,
		`"output": "present"`,
		"[DEBUG OpenAI Subscription Responses] event: response.output_text.delta",
	} {
		if !strings.Contains(debug, want) {
			t.Fatalf("debug output missing %q:\n%s", want, debug)
		}
	}
	for _, leaked := range []string{
		"system-prompt-secret",
		"user-prompt-secret",
		"followup-user-secret",
		"assistant-content-secret",
		"request-encrypted-secret",
		"request-argument-secret",
		"tool-output-secret",
		"tool-description-secret",
		"stream-encrypted-secret",
		"stream-text-secret",
		"SSE line:",
		"raw data:",
	} {
		if strings.Contains(debug, leaked) {
			t.Fatalf("debug output leaked %q:\n%s", leaked, debug)
		}
	}
}

func TestSubscriptionProviderBuildsFullPayloadReplayItems(t *testing.T) {
	provider := NewSubscription()
	assistant := api.Message{Role: "assistant", Content: "I'll inspect README."}
	assistant.SetOpenAIResponsesInputItems([]api.InputItem{
		{Type: "reasoning", ID: "rs_1", EncryptedContent: "encrypted-state"},
		{Type: "message", Role: "assistant", ID: "msg_1", Content: "I'll inspect README."},
		{Type: "function_call", ID: "fc_1", CallID: "call_1", Name: "read_file", Arguments: `{"path":"README.md"}`},
	})
	toolResult := api.Message{Role: "tool", Content: "README contents", ToolCallID: "call_1", ToolName: "read_file"}
	toolResult.SetOpenAIResponsesInputItems([]api.InputItem{
		{Type: "function_call_output", CallID: "call_1", Output: "README contents"},
	})

	req := provider.buildChatResponsesRequest(newOpenAITestContext(t, false), "system prompt", []api.Message{
		{Role: "user", Content: "inspect README"},
		assistant,
		toolResult,
		{Role: "user", Content: "now summarize"},
	}, "gpt-5.5")

	if req.PreviousResponseID != "" {
		t.Fatalf("PreviousResponseID = %q, want omitted", req.PreviousResponseID)
	}
	if len(req.ContextManagement) != 0 {
		t.Fatalf("ContextManagement = %#v, want omitted", req.ContextManagement)
	}
	input, ok := req.Input.([]api.InputItem)
	if !ok {
		t.Fatalf("Input type = %T, want []api.InputItem", req.Input)
	}
	if len(input) != 7 {
		t.Fatalf("len(Input) = %d, want developer + user + 3 assistant replay + tool output + current user: %#v", len(input), input)
	}
	if input[2].Type != "reasoning" || input[2].EncryptedContent != "encrypted-state" {
		t.Fatalf("Input[2] = %#v, want reasoning replay item", input[2])
	}
	if input[4].Type != "function_call" || input[4].CallID != "call_1" {
		t.Fatalf("Input[4] = %#v, want function_call replay item", input[4])
	}
	if input[5].Type != "function_call_output" || input[5].Output != "README contents" {
		t.Fatalf("Input[5] = %#v, want function_call_output replay item", input[5])
	}
	if input[6].Type != "message" || input[6].Role != "user" || input[6].Content != "now summarize" {
		t.Fatalf("Input[6] = %#v, want current user last", input[6])
	}
}

func TestSubscriptionProviderOmitsMaxOutputTokensEvenWhenConfigured(t *testing.T) {
	cfg := config.DefaultConfig()
	pCfg, _ := cfg.GetProviderModelConfig(subscriptionProviderKey)
	pCfg.DefaultModel = "gpt-5.5"
	pCfg.MaxOutputTokens = 64
	pCfg.ModelOverrides = map[string]config.ModelOverride{
		"gpt-5.5": {MaxOutputTokens: 128},
	}
	cfg.SetProviderModelConfig(subscriptionProviderKey, pCfg)

	provider := NewSubscription()
	req := provider.buildChatResponsesRequest(
		config.WithContext(newOpenAITestContext(t, false), cfg),
		"system prompt",
		[]api.Message{{Role: "user", Content: "hello"}},
		"gpt-5.5",
	)

	if req.MaxOutputTokens != 0 {
		t.Fatalf("MaxOutputTokens = %d, want omitted for subscription endpoint", req.MaxOutputTokens)
	}
}

func TestSubscriptionProviderCompactHistoryUsesOAuthTransportAndDecodesOutput(t *testing.T) {
	authDir := t.TempDir()
	t.Setenv(subscriptionAuthDirEnv, authDir)
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
			t.Fatalf("decode compact request: %v", err)
		}
		writeJSON(t, w, map[string]any{
			"model": "gpt-5.4-mini",
			"output": []map[string]any{{
				"type":              "reasoning",
				"id":                "rs_compact",
				"encrypted_content": "encrypted-compact-state",
			}, {
				"type": "compacted",
				"data": "compact-data",
			}},
			"usage": map[string]any{
				"input_tokens":  17,
				"output_tokens": 5,
				"total_tokens":  22,
			},
		})
	})
	t.Setenv(subscriptionCompactEndpointEnv, server.URL)
	if err := SaveSubscriptionCredential(DefaultSubscriptionAuthConfig(), SubscriptionCredential{
		AccessToken:  "oauth-access-token",
		RefreshToken: "oauth-refresh-token",
		AccountID:    "acct_1234abcd",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}

	resp, err := NewSubscription().CompactHistory(context.Background(), []api.InputItem{{Type: "message", Role: "user", Content: "compact me"}}, "", "compact instructions")
	if err != nil {
		t.Fatalf("CompactHistory() error = %v", err)
	}
	if authorization != "Bearer oauth-access-token" || strings.Contains(authorization, "platform-key") {
		t.Fatalf("Authorization = %q, want OAuth bearer and no OPENAI_API_KEY", authorization)
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
	if raw["model"] != "gpt-5.4-mini" {
		t.Fatalf("compact model = %#v, want default utility model", raw["model"])
	}
	if raw["instructions"] != "compact instructions" {
		t.Fatalf("compact instructions = %#v, want caller instructions", raw["instructions"])
	}
	input, ok := raw["input"].([]any)
	if !ok || len(input) != 1 {
		t.Fatalf("compact input = %#v, want one item", raw["input"])
	}
	if resp.Model != "gpt-5.4-mini" || len(resp.Output) != 2 {
		t.Fatalf("CompactHistory() response = %+v, want model and output", resp)
	}
	if resp.Output[0].EncryptedContent != "encrypted-compact-state" || resp.Output[1].Data != "compact-data" {
		t.Fatalf("CompactHistory() output = %#v, want rich compact items", resp.Output)
	}
	if resp.Usage == nil || resp.Usage.InputTokens != 17 || resp.Usage.OutputTokens != 5 {
		t.Fatalf("CompactHistory() usage = %+v, want decoded usage", resp.Usage)
	}
}

func TestSubscriptionProviderCompactHistoryRejectsUnsupportedModelBeforeNetwork(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir()+"/auth")
	var requests int
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	})
	t.Setenv(subscriptionCompactEndpointEnv, server.URL)

	_, err := NewSubscription().CompactHistory(context.Background(), nil, "gpt-5.2", "")
	if err == nil || !strings.Contains(err.Error(), "model gpt-5.2 is not supported by openai_subscription") {
		t.Fatalf("CompactHistory() error = %v, want unsupported model", err)
	}
	if requests != 0 {
		t.Fatalf("CompactHistory() sent %d requests, want 0 before model validation passes", requests)
	}
}

func TestSubscriptionProviderCompactHistoryRefusesOpenAIPlatformCompactEndpoint(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir()+"/auth")
	t.Setenv(subscriptionCompactEndpointEnv, openAIPlatformCompactURL)
	t.Setenv("OPENAI_API_KEY", "platform-key-must-not-be-used")

	_, err := NewSubscription().CompactHistory(context.Background(), []api.InputItem{{Type: "message", Role: "user", Content: "compact"}}, "gpt-5.4-mini", "")
	if err == nil || !strings.Contains(err.Error(), "must not use OpenAI Platform Compact API endpoint") {
		t.Fatalf("CompactHistory() error = %v, want forbidden Platform endpoint", err)
	}
}

func TestSubscriptionProviderCompactHistoryRefusesOpenAIPlatformCompactEndpointWithExplicitPort(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir()+"/auth")
	t.Setenv(subscriptionCompactEndpointEnv, "https://api.openai.com:443/v1/responses/compact")
	t.Setenv("OPENAI_API_KEY", "platform-key-must-not-be-used")

	_, err := NewSubscription().CompactHistory(context.Background(), []api.InputItem{{Type: "message", Role: "user", Content: "compact"}}, "gpt-5.4-mini", "")
	if err == nil || !strings.Contains(err.Error(), "must not use OpenAI Platform Compact API endpoint") {
		t.Fatalf("CompactHistory() error = %v, want forbidden Platform endpoint with explicit port", err)
	}
	if strings.Contains(err.Error(), "not logged in") {
		t.Fatalf("CompactHistory() error = %v, want platform endpoint validation before auth", err)
	}
}

func TestSubscriptionProviderCompactHistoryRejectsInvalidCompactEndpointBeforeAuth(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir()+"/auth")
	t.Setenv(subscriptionCompactEndpointEnv, "ftp://chatgpt.example.test/backend-api/codex/responses/compact")

	_, err := NewSubscription().CompactHistory(context.Background(), []api.InputItem{{Type: "message", Role: "user", Content: "compact"}}, "gpt-5.4-mini", "")
	if err == nil || !strings.Contains(err.Error(), "subscription Compact API endpoint must use http or https") {
		t.Fatalf("CompactHistory() error = %v, want invalid endpoint error before auth", err)
	}
	if strings.Contains(err.Error(), "not logged in") {
		t.Fatalf("CompactHistory() error = %v, want endpoint validation before auth", err)
	}
}

func TestSubscriptionProviderCompactHistoryRedactsHTTPErrorBody(t *testing.T) {
	authDir := t.TempDir()
	t.Setenv(subscriptionAuthDirEnv, authDir)
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"access_token":"access-secret-token","refresh_token":"refresh-secret-token","message":"bad compact token"}`))
	})
	t.Setenv(subscriptionCompactEndpointEnv, server.URL)
	if err := SaveSubscriptionCredential(DefaultSubscriptionAuthConfig(), SubscriptionCredential{
		AccessToken:  "oauth-access-token",
		RefreshToken: "oauth-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}

	_, err := NewSubscription().CompactHistory(context.Background(), []api.InputItem{{Type: "message", Role: "user", Content: "compact"}}, "gpt-5.4-mini", "")
	if err == nil {
		t.Fatal("CompactHistory() error = nil, want HTTP error")
	}
	for _, leaked := range []string{"access-secret-token", "refresh-secret-token"} {
		if strings.Contains(err.Error(), leaked) {
			t.Fatalf("CompactHistory() error leaked %q: %v", leaked, err)
		}
	}
}

func TestSubscriptionProviderCompactHistoryUsesLongRunningHTTPClient(t *testing.T) {
	authDir := t.TempDir()
	t.Setenv(subscriptionAuthDirEnv, authDir)
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(25 * time.Millisecond)
		writeJSON(t, w, map[string]any{
			"model": "gpt-5.4-mini",
			"output": []map[string]any{{
				"type": "compacted",
				"data": "compact-data",
			}},
		})
	})
	t.Setenv(subscriptionCompactEndpointEnv, server.URL)
	if err := SaveSubscriptionCredential(DefaultSubscriptionAuthConfig(), SubscriptionCredential{
		AccessToken:  "oauth-access-token",
		RefreshToken: "oauth-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}
	provider := NewSubscription()
	provider.HTTPClient = &http.Client{
		Timeout:   time.Second,
		Transport: &http.Transport{ResponseHeaderTimeout: time.Nanosecond},
	}

	resp, err := provider.CompactHistory(context.Background(), []api.InputItem{{Type: "message", Role: "user", Content: "compact"}}, "gpt-5.4-mini", "")
	if err != nil {
		t.Fatalf("CompactHistory() error = %v, want delayed headers to be governed by caller timeout", err)
	}
	if len(resp.Output) != 1 || resp.Output[0].Data != "compact-data" {
		t.Fatalf("CompactHistory() response = %+v, want compact output", resp)
	}
}

func TestSubscriptionProviderThinkingPolicyUsesSharedResponsesReasoning(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		configure  func(*config.Config)
		wantEffort string
	}{
		{
			name:  "thinking off omits reasoning for GPT",
			model: "gpt-5.5",
		},
		{
			name:  "thinking enabled sends selected effort",
			model: "gpt-5.4-mini",
			configure: func(cfg *config.Config) {
				cfg.Thinking.Enabled = true
				cfg.Thinking.Level = "high"
			},
			wantEffort: "high",
		},
		{
			name:       "codex spark keeps low fallback when thinking off",
			model:      "gpt-5.3-codex-spark",
			wantEffort: "low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			if tt.configure != nil {
				tt.configure(cfg)
			}
			req := NewSubscription().buildChatResponsesRequest(
				config.WithContext(context.Background(), cfg),
				"system prompt",
				[]api.Message{{Role: "user", Content: "hello"}},
				tt.model,
			)
			if tt.wantEffort == "" {
				if req.Reasoning != nil {
					t.Fatalf("Reasoning = %#v, want omitted", req.Reasoning)
				}
				return
			}
			if req.Reasoning == nil || req.Reasoning.Effort != tt.wantEffort {
				t.Fatalf("Reasoning = %#v, want effort %q", req.Reasoning, tt.wantEffort)
			}
		})
	}
}

func TestSubscriptionProviderRedactsHTTPErrorBody(t *testing.T) {
	authDir := t.TempDir()
	t.Setenv(subscriptionAuthDirEnv, authDir)
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"access_token":"access-secret-token","refresh_token":"refresh-secret-token","message":"bad token"}`))
	})
	t.Setenv(subscriptionEndpointEnv, server.URL)
	if err := SaveSubscriptionCredential(DefaultSubscriptionAuthConfig(), SubscriptionCredential{
		AccessToken:  "oauth-access-token",
		RefreshToken: "oauth-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}
	_, err := NewSubscription().ChatWithTools(newOpenAITestContext(t, false), "system", []api.Message{{Role: "user", Content: "hello"}}, "gpt-5.5")
	if err == nil {
		t.Fatal("ChatWithTools() error = nil, want HTTP error")
	}
	for _, leaked := range []string{"access-secret-token", "refresh-secret-token"} {
		if strings.Contains(err.Error(), leaked) {
			t.Fatalf("HTTP error leaked %q: %v", leaked, err)
		}
	}
}
