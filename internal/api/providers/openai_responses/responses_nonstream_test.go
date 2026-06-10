package openairesponses

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestHandleNonStreaming_CapturesReplayItemsInOutputOrder(t *testing.T) {
	resp := newResponsesNonStreamingHTTPResponse(`{
		"id": "resp_replay",
		"status": "completed",
		"output": [
			{
				"type": "reasoning",
				"id": "rs_1",
				"status": "completed",
				"summary": [{"text": "checked context"}],
				"encrypted_content": "encrypted-state"
			},
			{
				"type": "message",
				"id": "msg_1",
				"status": "completed",
				"content": [{"type":"output_text","text":"Need a file"}]
			},
			{
				"type": "function_call",
				"id": "fc_1",
				"status": "completed",
				"call_id": "call_1",
				"name": "read_file",
				"arguments": "{\"path\":\"README.md\"}"
			}
		]
	}`)
	defer resp.Body.Close()

	var replayItems []api.InputItem
	ctx := api.WithAssistantUpdateMode(context.Background(), api.AssistantUpdatesOff)
	content, responseID, err := HandleNonStreaming(ctx, resp, nil, NonStreamingOptions{
		ProviderName: "OpenAI",
		ReplayItemsCallback: func(items []api.InputItem) {
			replayItems = api.CloneInputItems(items)
		},
	})
	if err != nil {
		t.Fatalf("HandleNonStreaming() error = %v", err)
	}
	if responseID != "resp_replay" {
		t.Fatalf("responseID = %q, want resp_replay", responseID)
	}
	if !strings.Contains(content, "Need a file") || !strings.Contains(content, `"tool":"read_file"`) {
		t.Fatalf("content = %q, want text + internal tool JSON", content)
	}
	if len(replayItems) != 3 {
		t.Fatalf("len(replayItems) = %d, want 3: %#v", len(replayItems), replayItems)
	}
	if replayItems[0].Type != "reasoning" || replayItems[0].ID != "rs_1" || replayItems[0].Status != "completed" || replayItems[0].EncryptedContent != "encrypted-state" {
		t.Fatalf("reasoning replay item = %#v, want encrypted reasoning item", replayItems[0])
	}
	if len(replayItems[0].Summary) != 1 || replayItems[0].Summary[0]["text"] != "checked context" {
		t.Fatalf("reasoning summary = %#v, want provider summary", replayItems[0].Summary)
	}
	if replayItems[1].Type != "message" || replayItems[1].ID != "msg_1" || replayItems[1].Content != "Need a file" {
		t.Fatalf("message replay item = %#v, want output message", replayItems[1])
	}
	if replayItems[2].Type != "function_call" || replayItems[2].ID != "fc_1" || replayItems[2].CallID != "call_1" || replayItems[2].Name != "read_file" {
		t.Fatalf("function_call replay item = %#v, want read_file call", replayItems[2])
	}
}

func newResponsesNonStreamingHTTPResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
