package openai

import (
	"io"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestResponsesStreamState_HandleFunctionCallAdded_DoesNotControlSpinner(t *testing.T) {
	spinner := ui.NewSpinnerWithWriter(io.Discard)
	state := newResponsesStreamState(spinner, io.Discard)

	state.handleFunctionCallAdded(&ResponsesItem{
		Type:   "function_call",
		CallID: "call_1",
		Name:   "read_file",
	})

	if spinner.IsActive() {
		t.Fatal("handleFunctionCallAdded() should not start spinner")
	}
	if len(state.callOrder) != 1 || state.callOrder[0] != "call_1" {
		t.Fatalf("callOrder = %+v, want [call_1]", state.callOrder)
	}
}

func TestResponsesStreamState_ShowFunctionCallSpinner_StartsSpinner(t *testing.T) {
	spinner := ui.NewSpinnerWithWriter(io.Discard)
	state := newResponsesStreamState(spinner, io.Discard)

	state.showFunctionCallSpinner(&ResponsesItem{
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

	state.showFunctionCallSpinner(&ResponsesItem{
		Type: "message",
		Name: "ignored",
	})
	if spinner.IsActive() {
		t.Fatal("showFunctionCallSpinner() should ignore non-function_call item")
	}
}
