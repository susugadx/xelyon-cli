package openairesponses

import (
	"io"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestHandleChunk_DispatchesResponseCreated(t *testing.T) {
	state := newResponsesStreamState(nil, io.Discard)

	textDelta, done, err := state.handleChunk(StreamChunk{
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
		chunk       StreamChunk
		wantErrPart string
	}{
		{
			name: "error event",
			chunk: StreamChunk{
				Type:  "error",
				Error: &Error{Message: "quota exceeded"},
			},
			wantErrPart: "quota exceeded",
		},
		{
			name: "response failed event",
			chunk: StreamChunk{
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

	_, done, err := state.handleChunk(StreamChunk{
		Type: "response.output_item.added",
		Item: &Item{
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

	_, done, err = state.handleChunk(StreamChunk{
		Type: "response.function_call_arguments.done",
		Item: &Item{
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

	textDelta, done, err := state.handleChunk(StreamChunk{
		Type: "response.completed",
		Response: &ResponseMetadata{
			Usage: &Usage{
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

func TestHandleChunk_DispatchesOutputTextDelta(t *testing.T) {
	state := newResponsesStreamState(nil, io.Discard)

	textDelta, done, err := state.handleChunk(StreamChunk{
		Type:  "response.output_text.delta",
		Delta: "hello",
	}, "")
	if err != nil {
		t.Fatalf("handleChunk() error = %v, want nil", err)
	}
	if done {
		t.Fatal("handleChunk() done = true, want false")
	}
	if textDelta != "hello" {
		t.Fatalf("handleChunk() textDelta = %q, want %q", textDelta, "hello")
	}
}

func TestHandleChunk_FunctionCallAddedStartsSpinnerViaDisplayState(t *testing.T) {
	spinner := ui.NewSpinnerWithWriter(io.Discard)
	state := newResponsesStreamState(spinner, io.Discard)

	_, done, err := state.handleChunk(StreamChunk{
		Type: "response.output_item.added",
		Item: &Item{
			Type:   "function_call",
			CallID: "call_1",
			Name:   "read_file",
		},
	}, "")
	if err != nil {
		t.Fatalf("handleChunk() error = %v, want nil", err)
	}
	if done {
		t.Fatal("handleChunk() done = true, want false")
	}
	if !spinner.IsActive() {
		t.Fatal("handleChunk() should start spinner for function_call event")
	}
	spinner.Stop()
}

func TestHandleChunk_OutputItemAddedWithCompactionTypeDoesNotBreakStreamingParser(t *testing.T) {
	spinner := ui.NewSpinnerWithWriter(io.Discard)
	state := newResponsesStreamState(spinner, io.Discard)

	textDelta, done, err := state.handleChunk(StreamChunk{
		Type: "response.output_item.added",
		Item: &Item{
			Type: "compaction",
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
	if spinner.IsActive() {
		t.Fatal("spinner should not start for non-function_call output items")
	}
}

func TestHandleChunk_UnknownStreamingEventDoesNotBreakParser(t *testing.T) {
	state := newResponsesStreamState(nil, io.Discard)

	textDelta, done, err := state.handleChunk(StreamChunk{
		Type: "response.future_event.unknown",
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
}
