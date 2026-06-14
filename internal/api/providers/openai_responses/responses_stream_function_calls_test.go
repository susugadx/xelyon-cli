package openairesponses

import (
	"io"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
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

func TestResponsesStreamState_OpenAIResponsesReplayItemsPreserveProviderOutputOrder(t *testing.T) {
	state := newResponsesStreamState(nil, io.Discard)
	messageIndex := 0
	reasoningIndex := 1
	functionIndex := 2

	state.handleOutputItemDone(StreamChunk{
		Type:        "response.output_item.done",
		OutputIndex: &messageIndex,
		Item: &Item{
			Type:   "message",
			ID:     "msg_1",
			Status: "completed",
		},
	})
	state.handleTextDelta(StreamChunk{
		Type:        "response.output_text.delta",
		OutputIndex: &messageIndex,
		Delta:       "Need README",
	})
	state.handleOutputItemDone(StreamChunk{
		Type:        "response.output_item.done",
		OutputIndex: &reasoningIndex,
		Item: &Item{
			Type:             "reasoning",
			ID:               "rs_1",
			Status:           "completed",
			Summary:          []map[string]any{{"text": "checked context"}},
			EncryptedContent: "encrypted-state",
		},
	})
	state.handleOutputItemDone(StreamChunk{
		Type:        "response.output_item.done",
		OutputIndex: &functionIndex,
		Item: &Item{
			Type:      "function_call",
			ID:        "fc_1",
			CallID:    "call_1",
			Name:      "read_file",
			Status:    "completed",
			Arguments: `{"path":"README.md"}`,
		},
	})

	items := state.openAIResponsesReplayItems()
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3: %#v", len(items), items)
	}
	assertReplayItem(t, items[0], "message", "msg_1", "completed")
	if items[0].Role != "assistant" || items[0].Content != "Need README" {
		t.Fatalf("message item = %#v, want assistant text", items[0])
	}
	assertReplayItem(t, items[1], "reasoning", "rs_1", "completed")
	if items[1].EncryptedContent != "encrypted-state" || len(items[1].Summary) != 1 || items[1].Summary[0]["text"] != "checked context" {
		t.Fatalf("reasoning item = %#v, want summary and encrypted content", items[1])
	}
	assertReplayItem(t, items[2], "function_call", "fc_1", "completed")
	if items[2].CallID != "call_1" || items[2].Name != "read_file" || items[2].Arguments != `{"path":"README.md"}` {
		t.Fatalf("function call item = %#v, want read_file replay item", items[2])
	}
}

func TestResponsesStreamState_OpenAIResponsesReplayItemsUseStableFallbackOrder(t *testing.T) {
	state := newResponsesStreamState(nil, io.Discard)
	state.textOut.WriteString("legacy text")
	state.functionCalls["z_call"] = &responsesFunctionCallAccumulator{
		ID:     "fc_z",
		CallID: "z_call",
		Name:   "search_code",
	}
	state.functionCalls["a_call"] = &responsesFunctionCallAccumulator{
		ID:     "fc_a",
		CallID: "a_call",
		Name:   "read_file",
	}

	items := state.openAIResponsesReplayItems()
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want legacy message + 2 calls: %#v", len(items), items)
	}
	if items[0].Type != "message" || items[0].Content != "legacy text" {
		t.Fatalf("legacy message = %#v, want text output replay", items[0])
	}
	if items[1].CallID != "a_call" || items[2].CallID != "z_call" {
		t.Fatalf("fallback function call order = [%s %s], want sorted by call key", items[1].CallID, items[2].CallID)
	}
}

func TestResponsesStreamState_OpenAIResponsesReplayItemsAppendUnorderedFunctionsAfterKnownOrder(t *testing.T) {
	state := newResponsesStreamState(nil, io.Discard)
	firstIndex := 0
	state.handleFunctionCallAdded(&Item{
		Type:   "function_call",
		ID:     "fc_known",
		CallID: "known_call",
		Name:   "read_file",
	}, &firstIndex)
	state.functionCalls["a_late"] = &responsesFunctionCallAccumulator{
		ID:     "fc_late_a",
		CallID: "a_late",
		Name:   "search_code",
	}
	state.functionCalls["z_late"] = &responsesFunctionCallAccumulator{
		ID:     "fc_late_z",
		CallID: "z_late",
		Name:   "list_files",
	}

	items := state.openAIResponsesReplayItems()
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3 function calls: %#v", len(items), items)
	}
	if items[0].CallID != "known_call" || items[1].CallID != "a_late" || items[2].CallID != "z_late" {
		t.Fatalf("replay order = [%s %s %s], want known order then sorted fallback", items[0].CallID, items[1].CallID, items[2].CallID)
	}
}

func TestResponsesStreamState_AppendFunctionCallsToOutputUsesStableFallbackOrder(t *testing.T) {
	state := newResponsesStreamState(nil, io.Discard)
	state.functionCalls["z_call"] = &responsesFunctionCallAccumulator{
		CallID:    "z_call",
		Name:      "search_code",
		Arguments: strings.Builder{},
	}
	state.functionCalls["a_call"] = &responsesFunctionCallAccumulator{
		CallID: "a_call",
		Name:   "read_file",
	}
	state.functionCalls["a_call"].Arguments.WriteString(`{"path":"README.md"}`)
	state.functionCalls["z_call"].Arguments.WriteString(`{"query":"main"}`)

	state.appendFunctionCallsToOutput()

	output := state.toolCallsOut.String()
	first := strings.Index(output, `"tool":"read_file"`)
	second := strings.Index(output, `"tool":"search_code"`)
	if first < 0 || second < 0 || first > second {
		t.Fatalf("tool call output = %q, want read_file before search_code", output)
	}
}

func assertReplayItem(t *testing.T, item api.InputItem, wantType, wantID, wantStatus string) {
	t.Helper()
	if item.Type != wantType || item.ID != wantID || item.Status != wantStatus {
		t.Fatalf("item = %#v, want type=%s id=%s status=%s", item, wantType, wantID, wantStatus)
	}
}
