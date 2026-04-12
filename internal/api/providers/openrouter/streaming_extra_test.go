package openrouter

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func openRouterStreamingResponse(body string) *http.Response {
	return &http.Response{
		Body: io.NopCloser(strings.NewReader(body)),
	}
}

func TestHandleStreamingResponse_EmitsToolCallsAndUsage(t *testing.T) {
	p := New("test-key")
	var usage api.Usage
	p.SetUsageCallback(func(u api.Usage) {
		usage = u
	})

	ctx, _ := newOpenRouterTestContext(t, nil)
	body := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"Hello"}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":""}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"path\":\"/tmp/demo.txt\"}"}}]}}]}`,
		`data: {"choices":[],"usage":{"prompt_tokens":9,"completion_tokens":4,"prompt_tokens_details":{"cached_tokens":2}}}`,
		`data: [DONE]`,
	}, "\n\n") + "\n\n"

	got, err := p.handleStreamingResponse(ctx, openRouterStreamingResponse(body), ui.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleStreamingResponse() error = %v", err)
	}
	if !strings.Contains(got, "Hello") {
		t.Fatalf("result = %q, want streamed text", got)
	}
	if !strings.Contains(got, `"tool":"read_file"`) {
		t.Fatalf("result = %q, want tool call JSON", got)
	}
	if !strings.Contains(got, `"/tmp/demo.txt"`) {
		t.Fatalf("result = %q, want tool arguments", got)
	}
	if usage.InputTokens != 9 || usage.OutputTokens != 4 || usage.CachedInputTokens != 2 {
		t.Fatalf("usage = %+v, want input=9 output=4 cached=2", usage)
	}
}

func TestHandleStreamingResponse_ContextCanceled(t *testing.T) {
	p := New("test-key")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

	body := "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n"
	got, err := p.handleStreamingResponse(ctx, openRouterStreamingResponse(body), ui.NewSpinnerWithWriter(io.Discard))
	if err != context.Canceled {
		t.Fatalf("handleStreamingResponse() error = %v, want %v", err, context.Canceled)
	}
	if got != "" {
		t.Fatalf("handleStreamingResponse() = %q, want empty partial content before first chunk", got)
	}
}

func TestHandleClaudeStreamingResponse_ToolUseOnly(t *testing.T) {
	p := New("test-key")
	var usage api.Usage
	p.SetUsageCallback(func(u api.Usage) {
		usage = u
	})

	ctx, _ := newOpenRouterTestContext(t, nil)
	body := strings.Join([]string{
		`data: {"type":"message_start","message":{"usage":{"input_tokens":6,"cache_read_input_tokens":1}}}`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"read_file"}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"path\":\"/tmp/demo.txt\"}"}}`,
		`data: {"type":"content_block_stop","index":0}`,
		`data: {"type":"message_delta","usage":{"output_tokens":3}}`,
		`data: {"type":"message_stop"}`,
	}, "\n\n") + "\n\n"

	got, err := p.handleClaudeStreamingResponse(ctx, openRouterStreamingResponse(body), ui.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleClaudeStreamingResponse() error = %v", err)
	}
	if !strings.Contains(got, `"tool":"read_file"`) {
		t.Fatalf("result = %q, want tool JSON", got)
	}
	if !strings.Contains(got, `"/tmp/demo.txt"`) {
		t.Fatalf("result = %q, want tool arguments", got)
	}
	if usage.InputTokens != 7 || usage.OutputTokens != 3 || usage.CachedInputTokens != 1 {
		t.Fatalf("usage = %+v, want input=7 output=3 cached=1", usage)
	}
}
