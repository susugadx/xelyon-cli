package kimi

import (
	"io"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func TestHandleStreamingResponse_ContentReasoningToolCallsAndUsage(t *testing.T) {
	p := New("test-key")
	var gotUsage api.Usage
	p.SetUsageCallback(func(usage api.Usage) {
		gotUsage = usage
	})

	resp := newKimiSSEResponse(
		`{"choices":[{"delta":{"reasoning_content":"Think"}}]}`,
		`{"choices":[{"delta":{"reasoning_content":" deeply"}}]}`,
		`{"choices":[{"delta":{"content":"Answer "}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"pa"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"th\":\"main.go\"}"}}]}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
		`{"choices":[],"usage":{"prompt_tokens":12,"completion_tokens":8,"cached_tokens":5}}`,
	)
	defer resp.Body.Close()

	ctx, out, _ := newKimiTestContext(t, true)
	got, err := p.handleStreamingResponse(ctx, resp, uiruntime.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleStreamingResponse() error = %v", err)
	}
	if !strings.Contains(got, "Answer ") || !strings.Contains(got, `"tool":"read_file"`) || !strings.Contains(got, `"path":"main.go"`) {
		t.Fatalf("handleStreamingResponse() = %q, want text and tool JSON", got)
	}
	if p.LastReasoningContent() != "Think deeply" {
		t.Fatalf("LastReasoningContent() = %q, want Think deeply", p.LastReasoningContent())
	}
	if !strings.Contains(out.String(), "Think deeply") {
		t.Fatalf("reasoning output = %q, want streamed reasoning", out.String())
	}
	if gotUsage.InputTokens != 12 || gotUsage.OutputTokens != 8 || gotUsage.CachedInputTokens != 5 || gotUsage.ThinkingTokens != 0 {
		t.Fatalf("usage = %+v, want input=12 output=8 cached=5 thinking=0", gotUsage)
	}
}
