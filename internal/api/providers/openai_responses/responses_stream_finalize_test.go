package openairesponses

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
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

func TestHandleStreaming_CapturesReplayItems(t *testing.T) {
	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_1"}}`,
		``,
		`data: {"type":"response.output_text.delta","delta":"Need a file"}`,
		``,
		`data: {"type":"response.output_item.added","item":{"type":"function_call","call_id":"call_1","name":"read_file","status":"in_progress"}}`,
		``,
		`data: {"type":"response.function_call_arguments.delta","item":{"call_id":"call_1"},"delta":"{\"path\":\"README.md\"}"}`,
		``,
		`data: {"type":"response.completed","response":{"id":"resp_1","usage":{"input_tokens":10,"output_tokens":4}}}`,
		``,
	}, "\n")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	var replayItems []api.InputItem
	content, responseID, err := HandleStreaming(context.Background(), resp, nil, StreamingOptions{
		ProviderName: "OpenAI",
		DebugWriter:  io.Discard,
		ReplayItemsCallback: func(items []api.InputItem) {
			replayItems = api.CloneInputItems(items)
		},
	})
	if err != nil {
		t.Fatalf("HandleStreaming() error = %v", err)
	}
	if responseID != "resp_1" {
		t.Fatalf("responseID = %q, want resp_1", responseID)
	}
	if !strings.Contains(content, "Need a file") || !strings.Contains(content, `"tool":"read_file"`) {
		t.Fatalf("content = %q, want text + internal tool JSON", content)
	}
	if len(replayItems) != 2 {
		t.Fatalf("len(replayItems) = %d, want 2: %#v", len(replayItems), replayItems)
	}
	if replayItems[0].Type != "message" || replayItems[0].Role != "assistant" || replayItems[0].Content != "Need a file" {
		t.Fatalf("message replay item = %#v, want assistant message", replayItems[0])
	}
	if replayItems[1].Type != "function_call" || replayItems[1].CallID != "call_1" || replayItems[1].Arguments != `{"path":"README.md"}` {
		t.Fatalf("function_call replay item = %#v, want read_file call", replayItems[1])
	}
}

func TestHandleStreaming_RoutesParallelFunctionArgumentsByItemID(t *testing.T) {
	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_parallel"}}`,
		``,
		`data: {"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"fc_read","call_id":"call_read","name":"read_file","status":"in_progress"}}`,
		``,
		`data: {"type":"response.output_item.added","output_index":1,"item":{"type":"function_call","id":"fc_search","call_id":"call_search","name":"search_code","status":"in_progress"}}`,
		``,
		`data: {"type":"response.function_call_arguments.delta","item_id":"fc_search","output_index":1,"delta":"{\"query\":\"main\"}"}`,
		``,
		`data: {"type":"response.function_call_arguments.delta","item_id":"fc_read","output_index":0,"delta":"{\"path\":\"README.md\"}"}`,
		``,
		`data: {"type":"response.completed","response":{"id":"resp_parallel","usage":{"input_tokens":10,"output_tokens":4}}}`,
		``,
	}, "\n")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	var replayItems []api.InputItem
	content, responseID, err := HandleStreaming(context.Background(), resp, nil, StreamingOptions{
		ProviderName: "OpenAI",
		DebugWriter:  io.Discard,
		ReplayItemsCallback: func(items []api.InputItem) {
			replayItems = api.CloneInputItems(items)
		},
	})
	if err != nil {
		t.Fatalf("HandleStreaming() error = %v", err)
	}
	if responseID != "resp_parallel" {
		t.Fatalf("responseID = %q, want resp_parallel", responseID)
	}
	for _, want := range []string{`"tool":"read_file"`, `"tool":"search_code"`} {
		if !strings.Contains(content, want) {
			t.Fatalf("content = %q, want %s", content, want)
		}
	}
	if len(replayItems) != 2 {
		t.Fatalf("len(replayItems) = %d, want 2: %#v", len(replayItems), replayItems)
	}
	if replayItems[0].CallID != "call_read" || replayItems[0].Arguments != `{"path":"README.md"}` {
		t.Fatalf("first replay item = %#v, want read_file args", replayItems[0])
	}
	if replayItems[1].CallID != "call_search" || replayItems[1].Arguments != `{"query":"main"}` {
		t.Fatalf("second replay item = %#v, want search_code args", replayItems[1])
	}
}

func TestHandleStreaming_CapturesReasoningReplayItemsInOutputOrder(t *testing.T) {
	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_1"}}`,
		``,
		`data: {"type":"response.output_item.added","output_index":0,"item":{"type":"reasoning","id":"rs_1","status":"completed","summary":[{"type":"summary_text","text":"checked files"}],"encrypted_content":"encrypted-state"}}`,
		``,
		`data: {"type":"response.output_item.added","output_index":1,"item":{"type":"message","id":"msg_1","status":"in_progress"}}`,
		``,
		`data: {"type":"response.output_text.delta","output_index":1,"delta":"Need a file"}`,
		``,
		`data: {"type":"response.output_item.added","output_index":2,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"read_file","status":"in_progress"}}`,
		``,
		`data: {"type":"response.function_call_arguments.done","item":{"call_id":"call_1","arguments":"{\"path\":\"README.md\"}"}}`,
		``,
		`data: {"type":"response.completed","response":{"id":"resp_1","usage":{"input_tokens":10,"output_tokens":4}}}`,
		``,
	}, "\n")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	var replayItems []api.InputItem
	_, _, err := HandleStreaming(context.Background(), resp, nil, StreamingOptions{
		ProviderName: "OpenAI",
		DebugWriter:  io.Discard,
		ReplayItemsCallback: func(items []api.InputItem) {
			replayItems = api.CloneInputItems(items)
		},
	})
	if err != nil {
		t.Fatalf("HandleStreaming() error = %v", err)
	}
	if len(replayItems) != 3 {
		t.Fatalf("len(replayItems) = %d, want reasoning + message + function_call: %#v", len(replayItems), replayItems)
	}
	if replayItems[0].Type != "reasoning" || replayItems[0].ID != "rs_1" || replayItems[0].Status != "completed" || replayItems[0].EncryptedContent != "encrypted-state" {
		t.Fatalf("reasoning replay item = %#v, want encrypted reasoning item", replayItems[0])
	}
	if len(replayItems[0].Summary) != 1 || replayItems[0].Summary[0]["text"] != "checked files" {
		t.Fatalf("reasoning summary = %#v, want provider summary", replayItems[0].Summary)
	}
	if replayItems[1].Type != "message" || replayItems[1].ID != "msg_1" || replayItems[1].Content != "Need a file" {
		t.Fatalf("message replay item = %#v, want assistant text in output order", replayItems[1])
	}
	if replayItems[2].Type != "function_call" || replayItems[2].ID != "fc_1" || replayItems[2].CallID != "call_1" || replayItems[2].Arguments != `{"path":"README.md"}` {
		t.Fatalf("function_call replay item = %#v, want call in output order", replayItems[2])
	}
}
