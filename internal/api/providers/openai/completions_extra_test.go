package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func newOpenAITestContext(t *testing.T, thinking bool) context.Context {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = thinking
	cfg.Thinking.Level = "high"
	runtime := uiruntime.NewRuntime(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	ctx := uiruntime.WithRuntime(context.Background(), runtime)
	ctx = config.WithContext(ctx, cfg)
	return api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
}

func newOpenAISSEResponse(chunks ...string) *http.Response {
	var body strings.Builder
	for _, chunk := range chunks {
		body.WriteString("data: ")
		body.WriteString(chunk)
		body.WriteString("\n\n")
	}
	body.WriteString("data: [DONE]\n\n")
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(body.String())),
	}
}

func TestProvider_IsFunctionCallingEnabled(t *testing.T) {
	t.Run("env unset", func(t *testing.T) {
		unsetOpenAIFunctionCallingEnv(t)
		if !New("test-key").IsFunctionCallingEnabled() {
			t.Fatal("IsFunctionCallingEnabled() = false, want true when OPENAI_FUNCTION_CALLING is unset")
		}
	})
	t.Run("enabled", func(t *testing.T) {
		t.Setenv("OPENAI_FUNCTION_CALLING", "1")
		if !New("test-key").IsFunctionCallingEnabled() {
			t.Fatal("IsFunctionCallingEnabled() = false, want true when OPENAI_FUNCTION_CALLING=1")
		}
	})
	t.Run("disabled", func(t *testing.T) {
		t.Setenv("OPENAI_FUNCTION_CALLING", "0")
		if New("test-key").IsFunctionCallingEnabled() {
			t.Fatal("IsFunctionCallingEnabled() = true, want false when OPENAI_FUNCTION_CALLING=0")
		}
	})
}

func unsetOpenAIFunctionCallingEnv(t *testing.T) {
	t.Helper()
	oldValue, hadValue := os.LookupEnv("OPENAI_FUNCTION_CALLING")
	if err := os.Unsetenv("OPENAI_FUNCTION_CALLING"); err != nil {
		t.Fatalf("unset OPENAI_FUNCTION_CALLING: %v", err)
	}
	t.Cleanup(func() {
		if hadValue {
			if err := os.Setenv("OPENAI_FUNCTION_CALLING", oldValue); err != nil {
				t.Fatalf("restore OPENAI_FUNCTION_CALLING: %v", err)
			}
			return
		}
		if err := os.Unsetenv("OPENAI_FUNCTION_CALLING"); err != nil {
			t.Fatalf("restore OPENAI_FUNCTION_CALLING: %v", err)
		}
	})
}

func TestHandleStreamingResponse_CombinesToolCallsAndUsage(t *testing.T) {
	p := New("test-key")
	var gotUsage api.Usage
	p.SetUsageCallback(func(usage api.Usage) {
		gotUsage = usage
	})

	resp := newOpenAISSEResponse(
		`{"choices":[{"delta":{"content":"Let me check."}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"pa"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"th\":\"main.go\"}"}}]}}]}`,
		`{"usage":{"prompt_tokens":10,"completion_tokens":4,"prompt_tokens_details":{"cached_tokens":3},"completion_tokens_details":{"reasoning_tokens":2}}}`,
		`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
	)
	defer resp.Body.Close()

	got, err := p.handleStreamingResponse(newOpenAITestContext(t, false), resp, uiruntime.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleStreamingResponse() error = %v", err)
	}
	if !strings.Contains(got, "Let me check.") || !strings.Contains(got, "read_file") || !strings.Contains(got, "main.go") {
		t.Fatalf("handleStreamingResponse() = %q, want text and tool JSON", got)
	}
	if gotUsage.InputTokens != 10 || gotUsage.OutputTokens != 2 || gotUsage.CachedInputTokens != 3 || gotUsage.ThinkingTokens != 2 {
		t.Fatalf("usage = %+v, want input=10 output=2 cached=3 thinking=2", gotUsage)
	}
}

func TestHandleStreamingResponse_NullUsageDoesNotEmitZeroUsageOnToolCallsFinish(t *testing.T) {
	p := New("test-key")
	callbackCount := 0
	p.SetUsageCallback(func(api.Usage) {
		callbackCount++
	})

	resp := newOpenAISSEResponse(
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"main.go\"}"}}]}}]}`,
		`{"choices":[],"usage":null}`,
		`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
	)
	defer resp.Body.Close()

	got, err := p.handleStreamingResponse(newOpenAITestContext(t, false), resp, uiruntime.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleStreamingResponse() error = %v", err)
	}
	if !strings.Contains(got, `"tool":"read_file"`) {
		t.Fatalf("handleStreamingResponse() = %q, want tool JSON", got)
	}
	if callbackCount != 0 {
		t.Fatalf("usage callback count = %d, want 0 when usage is null only", callbackCount)
	}
}

func TestChatWithCompletions_RequestIncludesThinkingAndForcedToolChoice(t *testing.T) {
	t.Setenv("OPENAI_FUNCTION_CALLING", "1")

	var captured openaicompat.ChatCompletionsRequest
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		streamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
	})
	t.Setenv("OPENAI_API_URL", server.URL)

	p := New("test-key")
	p.SetMCPTools([]api.ToolDefinition{{
		Name:        "custom_lookup",
		Description: "custom lookup",
		Parameters:  map[string]any{"type": "object"},
	}})
	p.SetToolChoice("custom_lookup")

	_, err := p.chatWithCompletions(newOpenAITestContext(t, true), "System", []api.Message{{Role: "user", Content: "Hello"}}, "gpt-4-turbo")
	if err != nil {
		t.Fatalf("chatWithCompletions() error = %v", err)
	}
	if captured.ReasoningEffort != "high" {
		t.Fatalf("ReasoningEffort = %q, want %q", captured.ReasoningEffort, "high")
	}
	if len(captured.Tools) == 0 {
		t.Fatal("Tools should be included when function calling is enabled")
	}
	toolChoice, ok := captured.ToolChoice.(map[string]any)
	if !ok {
		t.Fatalf("ToolChoice type = %T, want map[string]any", captured.ToolChoice)
	}
	function, ok := toolChoice["function"].(map[string]any)
	if !ok || function["name"] != "custom_lookup" {
		t.Fatalf("ToolChoice function = %#v, want custom_lookup", toolChoice["function"])
	}
}

func TestChatWithCompletions_FunctionCallingDisabledOmitsToolFields(t *testing.T) {
	t.Setenv("OPENAI_FUNCTION_CALLING", "0")

	var captured map[string]any
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		streamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
	})
	t.Setenv("OPENAI_API_URL", server.URL)

	p := New("test-key")
	p.SetMCPTools([]api.ToolDefinition{{Name: "custom_lookup", Description: "custom lookup"}})
	p.SetToolChoice("custom_lookup")

	if _, err := p.chatWithCompletions(newOpenAITestContext(t, false), "System", []api.Message{{Role: "user", Content: "Hello"}}, "gpt-4-turbo"); err != nil {
		t.Fatalf("chatWithCompletions() error = %v", err)
	}
	if _, ok := captured["tools"]; ok {
		t.Fatalf("tools should be omitted when function calling is disabled: %#v", captured["tools"])
	}
	if _, ok := captured["tool_choice"]; ok {
		t.Fatalf("tool_choice should be omitted when function calling is disabled: %#v", captured["tool_choice"])
	}
}

func TestChatWithCompletions_PromptCacheScopeDoesNotChangeOpenAIKey(t *testing.T) {
	var captured openaicompat.ChatCompletionsRequest
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		streamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
	})
	t.Setenv("OPENAI_API_URL", server.URL)
	t.Setenv("OPENAI_FUNCTION_CALLING", "0")

	ctx := api.WithPromptCacheScope(newOpenAITestContext(t, false), api.PromptCacheScope{SessionID: "session-1"})
	_, err := New("test-key").chatWithCompletions(ctx, "System", []api.Message{{Role: "user", Content: "Hello"}}, "gpt-4-turbo")
	if err != nil {
		t.Fatalf("chatWithCompletions() error = %v", err)
	}

	want := BuildPromptCacheKey("gpt-4-turbo", "System")
	if captured.PromptCacheKey != want {
		t.Fatalf("PromptCacheKey = %q, want %q", captured.PromptCacheKey, want)
	}
	if !strings.HasPrefix(captured.PromptCacheKey, "xelyon:v2:") {
		t.Fatalf("PromptCacheKey = %q, want existing xelyon:v2 format", captured.PromptCacheKey)
	}
}

func TestBuildChatCompletionsRequest_IncludesActiveContextFromContext(t *testing.T) {
	ctx := api.WithActiveContextBlocks(newOpenAITestContext(t, false), openAITestActiveContextBlocks())

	req := New("test-key").buildChatCompletionsRequest(
		ctx,
		"System",
		[]api.Message{{Role: "user", Content: "Hello"}},
		"gpt-4-turbo",
	)

	if len(req.Messages) != 3 {
		t.Fatalf("len(Messages) = %d, want system + active context + history", len(req.Messages))
	}
	if req.Messages[0].Role != "system" || req.Messages[0].Content != "System" {
		t.Fatalf("Messages[0] = %#v, want system message", req.Messages[0])
	}
	if req.Messages[1].Role != "system" || req.Messages[1].Content != openAITestActiveContextSnapshot {
		t.Fatalf("Messages[1] = %#v, want active context system message", req.Messages[1])
	}
	if req.Messages[2].Role != "user" || req.Messages[2].Content != "Hello" {
		t.Fatalf("Messages[2] = %#v, want original history message", req.Messages[2])
	}
}

func TestChatWithCompletions_ToolUseDisabledOmitsToolFields(t *testing.T) {
	t.Setenv("OPENAI_FUNCTION_CALLING", "1")

	var captured map[string]any
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		streamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
	})
	t.Setenv("OPENAI_API_URL", server.URL)

	p := New("test-key")
	p.SetMCPTools([]api.ToolDefinition{{Name: "custom_lookup", Description: "custom lookup"}})
	p.SetToolChoice("custom_lookup")

	ctx := api.WithToolUseDisabled(newOpenAITestContext(t, false))
	if _, err := p.chatWithCompletions(ctx, "System", []api.Message{{Role: "user", Content: "Hello"}}, "gpt-4-turbo"); err != nil {
		t.Fatalf("chatWithCompletions() error = %v", err)
	}
	if _, ok := captured["tools"]; ok {
		t.Fatalf("tools should be omitted when tool use is disabled: %#v", captured["tools"])
	}
	if _, ok := captured["tool_choice"]; ok {
		t.Fatalf("tool_choice should be omitted when tool use is disabled: %#v", captured["tool_choice"])
	}
}
