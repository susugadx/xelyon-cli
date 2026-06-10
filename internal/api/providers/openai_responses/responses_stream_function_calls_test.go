package openairesponses

import (
	"io"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestResponsesStreamState_HandleFunctionCallAdded_DoesNotControlSpinner(t *testing.T) {
	spinner := ui.NewSpinnerWithWriter(io.Discard)
	state := newResponsesStreamState(spinner, io.Discard)

	state.handleFunctionCallAdded(&Item{
		Type:   "function_call",
		CallID: "call_1",
		Name:   "read_file",
	}, nil)

	if spinner.IsActive() {
		t.Fatal("handleFunctionCallAdded() should not start spinner")
	}
	if len(state.callOrder) != 1 || state.callOrder[0] != "call_1" {
		t.Fatalf("callOrder = %+v, want [call_1]", state.callOrder)
	}
}

func TestResponsesStreamState_FunctionCallArgumentsUseTopLevelItemID(t *testing.T) {
	state := newResponsesStreamState(nil, io.Discard)
	firstIndex := 0
	secondIndex := 1

	state.handleFunctionCallAdded(&Item{
		Type:   "function_call",
		ID:     "fc_read",
		CallID: "call_read",
		Name:   "read_file",
	}, &firstIndex)
	state.handleFunctionCallAdded(&Item{
		Type:   "function_call",
		ID:     "fc_search",
		CallID: "call_search",
		Name:   "search_code",
	}, &secondIndex)

	state.handleFunctionCallArgumentsDelta(StreamChunk{
		Type:        "response.function_call_arguments.delta",
		ItemID:      "fc_search",
		OutputIndex: &secondIndex,
		Delta:       `{"query":"main"}`,
	})
	state.handleFunctionCallArgumentsDelta(StreamChunk{
		Type:        "response.function_call_arguments.delta",
		ItemID:      "fc_read",
		OutputIndex: &firstIndex,
		Delta:       `{"path":"README.md"}`,
	})

	if got := state.functionCalls["call_read"].Arguments.String(); got != `{"path":"README.md"}` {
		t.Fatalf("call_read arguments = %q, want README path", got)
	}
	if got := state.functionCalls["call_search"].Arguments.String(); got != `{"query":"main"}` {
		t.Fatalf("call_search arguments = %q, want query", got)
	}
}

func TestResponsesStreamState_FunctionCallArgumentsDoneUsesTopLevelArguments(t *testing.T) {
	state := newResponsesStreamState(nil, io.Discard)
	outputIndex := 0
	state.handleFunctionCallAdded(&Item{
		Type:   "function_call",
		ID:     "fc_search",
		CallID: "call_search",
		Name:   "search_code",
	}, &outputIndex)

	state.handleFunctionCallArgumentsDelta(StreamChunk{
		Type:        "response.function_call_arguments.delta",
		ItemID:      "fc_search",
		OutputIndex: &outputIndex,
		Delta:       `{"query":"partial"}`,
	})
	state.handleFunctionCallArgumentsDone(StreamChunk{
		Type:        "response.function_call_arguments.done",
		ItemID:      "fc_search",
		OutputIndex: &outputIndex,
		Arguments:   `{"query":"final"}`,
	})

	if got := state.functionCalls["call_search"].Arguments.String(); got != `{"query":"final"}` {
		t.Fatalf("call_search arguments = %q, want final top-level arguments", got)
	}
}

func TestResponsesStreamState_ShowFunctionCallSpinner_StartsSpinner(t *testing.T) {
	spinner := ui.NewSpinnerWithWriter(io.Discard)
	state := newResponsesStreamState(spinner, io.Discard)

	state.showFunctionCallSpinner(&Item{
		Type: "function_call",
		Name: "read_file",
	})
	if !spinner.IsActive() {
		t.Fatal("showFunctionCallSpinner() should start spinner for function_call")
	}
	spinner.Stop()
}

func TestResponsesStreamState_ShowFunctionCallSpinner_IgnoresNonFunctionCall(t *testing.T) {
	spinner := ui.NewSpinnerWithWriter(io.Discard)
	state := newResponsesStreamState(spinner, io.Discard)

	state.showFunctionCallSpinner(&Item{
		Type: "message",
		Name: "ignored",
	})
	if spinner.IsActive() {
		t.Fatal("showFunctionCallSpinner() should ignore non-function_call item")
	}
}
