package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newOpenAITestContext(t *testing.T, thinking bool) context.Context {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = thinking
	cfg.Thinking.Level = "high"
	runtime := ui.NewRuntime(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	ctx := ui.WithRuntime(context.Background(), runtime)
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

func TestProvider_IsFunctionCallingEnabledAlwaysTrue(t *testing.T) {
	if !New("test-key").IsFunctionCallingEnabled() {
		t.Fatal("IsFunctionCallingEnabled() should always be true for OpenAI provider")
	}
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

	got, err := p.handleStreamingResponse(newOpenAITestContext(t, false), resp, ui.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleStreamingResponse() error = %v", err)
	}
	if !strings.Contains(got, "Let me check.") || !strings.Contains(got, "read_file") || !strings.Contains(got, "main.go") {
		t.Fatalf("handleStreamingResponse() = %q, want text and tool JSON", got)
	}
	if gotUsage.InputTokens != 10 || gotUsage.OutputTokens != 4 || gotUsage.CachedInputTokens != 3 || gotUsage.ThinkingTokens != 2 {
		t.Fatalf("usage = %+v, want input=10 output=4 cached=3 thinking=2", gotUsage)
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

	got, err := p.handleStreamingResponse(newOpenAITestContext(t, false), resp, ui.NewSpinnerWithWriter(io.Discard))
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
	var captured api.ChatRequest
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
