package openai

import (
	"errors"
	"io"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestResponsesStreamFinalizePolicy_FinalizeParseError(t *testing.T) {
	state := newResponsesStreamState(nil, io.Discard)
	state.responseID = "resp_123"
	parseErr := errors.New("parse failed")

	content, responseID, err := newResponsesStreamFinalizePolicy(state, nil).finalize("partial", parseErr)
	if !errors.Is(err, parseErr) {
		t.Fatalf("finalize() err = %v, want %v", err, parseErr)
	}
	if content != "" {
		t.Fatalf("finalize() content = %q, want empty on parse error", content)
	}
	if responseID != "resp_123" {
		t.Fatalf("finalize() responseID = %q, want %q", responseID, "resp_123")
	}
}

func TestResponsesStreamFinalizePolicy_FinalizeEmitsUsageAndAppendsToolCalls(t *testing.T) {
	state := newResponsesStreamState(nil, io.Discard)
	state.responseID = "resp_123"
	state.lastUsage = &api.Usage{InputTokens: 10, OutputTokens: 5}
	state.toolCallsOut.WriteString(`{"tool":"read_file"}`)

	var gotUsage api.Usage
	content, responseID, err := newResponsesStreamFinalizePolicy(state, func(u api.Usage) {
		gotUsage = u
	}).finalize("Hello ", nil)
	if err != nil {
		t.Fatalf("finalize() err = %v, want nil", err)
	}
	if content != `Hello {"tool":"read_file"}` {
		t.Fatalf("finalize() content = %q, want merged content and tool call", content)
	}
	if responseID != "resp_123" {
		t.Fatalf("finalize() responseID = %q, want %q", responseID, "resp_123")
	}
	if gotUsage.InputTokens != 10 || gotUsage.OutputTokens != 5 {
		t.Fatalf("usage callback = %+v, want input=10 output=5", gotUsage)
	}
}

func TestResponsesStreamFinalizePolicy_FinalizeToolCallsOnly(t *testing.T) {
	state := newResponsesStreamState(nil, io.Discard)
	state.toolCallsOut.WriteString(`{"tool":"read_file"}`)

	content, _, err := newResponsesStreamFinalizePolicy(state, nil).finalize("", nil)
	if err != nil {
		t.Fatalf("finalize() err = %v, want nil", err)
	}
	if content != `{"tool":"read_file"}` {
		t.Fatalf("finalize() content = %q, want tool call only", content)
	}
}
