package openai

import (
	"io"
	"strings"
	"testing"
)

func TestHandleChunk_DispatchesResponseCreated(t *testing.T) {
	state := newResponsesStreamState(nil, io.Discard)

	textDelta, done, err := state.handleChunk(ResponsesStreamChunk{
		Type: "response.created",
		Response: &ResponseMetadata{
			ID: "resp_123",
		},
	}, "")
	if err != nil {
		t.Fatalf("handleChunk() error = %v, want nil", err)
	}
	if done {
		t.Fatal("handleChunk() done = true, want false")
	}
	if textDelta != "" {
		t.Fatalf("handleChunk() textDelta = %q, want empty", textDelta)
	}
	if state.responseID != "resp_123" {
		t.Fatalf("responseID = %q, want %q", state.responseID, "resp_123")
	}
}

func TestHandleChunk_DispatchesErrorEvents(t *testing.T) {
	tests := []struct {
		name        string
		chunk       ResponsesStreamChunk
		wantErrPart string
	}{
		{
			name: "error event",
			chunk: ResponsesStreamChunk{
				Type:  "error",
				Error: &ResponsesError{Message: "quota exceeded"},
			},
			wantErrPart: "quota exceeded",
		},
		{
			name: "response failed event",
			chunk: ResponsesStreamChunk{
				Type: "response.failed",
			},
			wantErrPart: "OpenAI Responses API request failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newResponsesStreamState(nil, io.Discard)
			textDelta, done, err := state.handleChunk(tt.chunk, "")
			if !done {
				t.Fatal("handleChunk() done = false, want true")
			}
			if textDelta != "" {
				t.Fatalf("handleChunk() textDelta = %q, want empty", textDelta)
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErrPart) {
				t.Fatalf("handleChunk() error = %v, want %q", err, tt.wantErrPart)
			}
		})
	}
}

func TestHandleChunk_DispatchesFunctionCallAndCompletion(t *testing.T) {
	state := newResponsesStreamState(nil, io.Discard)

	_, done, err := state.handleChunk(ResponsesStreamChunk{
		Type: "response.output_item.added",
		Item: &ResponsesItem{
			Type:   "function_call",
			CallID: "call_1",
			Name:   "read_file",
		},
	}, "")
	if err != nil {
		t.Fatalf("handleChunk(add) error = %v, want nil", err)
	}
	if done {
		t.Fatal("handleChunk(add) done = true, want false")
	}

	_, done, err = state.handleChunk(ResponsesStreamChunk{
		Type: "response.function_call_arguments.done",
		Item: &ResponsesItem{
			CallID:    "call_1",
			Arguments: `{"path":"main.go"}`,
		},
	}, "")
	if err != nil {
		t.Fatalf("handleChunk(arguments.done) error = %v, want nil", err)
	}
	if done {
		t.Fatal("handleChunk(arguments.done) done = true, want false")
	}

	textDelta, done, err := state.handleChunk(ResponsesStreamChunk{
		Type: "response.completed",
		Response: &ResponseMetadata{
			Usage: &ResponsesUsage{
				InputTokens:  10,
				OutputTokens: 4,
			},
		},
	}, "")
	if err != nil {
		t.Fatalf("handleChunk(completed) error = %v, want nil", err)
	}
	if !done {
		t.Fatal("handleChunk(completed) done = false, want true")
	}
	if textDelta != "" {
		t.Fatalf("handleChunk(completed) textDelta = %q, want empty", textDelta)
	}
	if !strings.Contains(state.toolCallsOut.String(), `"tool":"read_file"`) {
		t.Fatalf("toolCallsOut = %q, want read_file tool JSON", state.toolCallsOut.String())
	}
	if state.lastUsage == nil || state.lastUsage.InputTokens != 10 || state.lastUsage.OutputTokens != 4 {
		t.Fatalf("lastUsage = %+v, want input=10 output=4", state.lastUsage)
	}
}
