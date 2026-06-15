package openairesponses

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
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

func TestHandleNonStreaming_EmitsUsageAndFallsBackToOutputTextReplay(t *testing.T) {
	resp := newResponsesNonStreamingHTTPResponse(`{
		"id": "resp_usage",
		"status": "completed",
		"output_text": "plain response",
		"usage": {
			"input_tokens": 11,
			"output_tokens": 7,
			"input_tokens_details": {"cached_tokens": 3},
			"output_tokens_details": {"reasoning_tokens": 2}
		}
	}`)
	defer resp.Body.Close()

	var gotUsage api.Usage
	var gotReplay []api.InputItem
	ctx := api.WithAssistantUpdateMode(context.Background(), api.AssistantUpdatesOff)
	content, responseID, err := HandleNonStreaming(ctx, resp, nil, NonStreamingOptions{
		UsageCallback: func(usage api.Usage) {
			gotUsage = usage
		},
		ReplayItemsCallback: func(items []api.InputItem) {
			gotReplay = api.CloneInputItems(items)
		},
	})
	if err != nil {
		t.Fatalf("HandleNonStreaming() error = %v", err)
	}
	if responseID != "resp_usage" || content != "plain response" {
		t.Fatalf("HandleNonStreaming() = (%q, %q), want plain response and resp_usage", content, responseID)
	}
	if gotUsage.InputTokens != 11 || gotUsage.OutputTokens != 5 || gotUsage.CachedInputTokens != 3 || gotUsage.ThinkingTokens != 2 {
		t.Fatalf("usage = %#v, want input/output/cache/reasoning tokens", gotUsage)
	}
	if len(gotReplay) != 1 || gotReplay[0].Type != "message" || gotReplay[0].Content != "plain response" {
		t.Fatalf("replay items = %#v, want output_text fallback message", gotReplay)
	}
}

func TestResponsesNonStreamingResponseValidateSuccessReportsProviderErrors(t *testing.T) {
	tests := []struct {
		name    string
		result  responsesNonStreamingResponse
		wantErr string
	}{
		{
			name:    "error message wins",
			result:  responsesNonStreamingResponse{Status: "failed", Error: &Error{Message: "bad request", Code: "bad_code", Type: "invalid_request_error"}},
			wantErr: "bad request",
		},
		{
			name:    "error code fallback",
			result:  responsesNonStreamingResponse{Error: &Error{Code: "bad_code", Type: "invalid_request_error"}},
			wantErr: "Provider API error: bad_code",
		},
		{
			name:    "error type fallback",
			result:  responsesNonStreamingResponse{Error: &Error{Type: "invalid_request_error"}},
			wantErr: "Provider API error: invalid_request_error",
		},
		{
			name:    "failed status without detail",
			result:  responsesNonStreamingResponse{Status: "failed"},
			wantErr: "Provider Responses API request failed",
		},
		{
			name:    "other noncompleted status",
			result:  responsesNonStreamingResponse{Status: "incomplete"},
			wantErr: "Provider Responses API response status: incomplete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.validateSuccess("Provider")
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("validateSuccess() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestRunResponsesRequest_RetriesOnceWithoutPreviousResponseIDForNonStreaming(t *testing.T) {
	previousResponseID := "resp_old"
	executeCalls := 0
	var payloads []string
	var observed []Request

	content, responseID, err := RunResponsesRequest(context.Background(), RunOptions{
		URL: "https://example.test/v1/responses",
		BuildRequest: func() Request {
			return Request{
				Model:              "gpt-5.4",
				Input:              "hello",
				PreviousResponseID: previousResponseID,
				Stream:             false,
				Store:              true,
			}
		},
		PrepareRequest: func(ctx context.Context, url string, payload []byte) (*http.Request, error) {
			payloads = append(payloads, string(payload))
			return http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(payload)))
		},
		ExecuteRequest: func(req *http.Request, stream bool) (*http.Response, error) {
			if stream {
				t.Fatal("ExecuteRequest() stream = true, want non-streaming")
			}
			executeCalls++
			if executeCalls == 1 {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader("bad previous_response_id")),
				}, nil
			}
			return newResponsesNonStreamingHTTPResponse(`{"id":"resp_new","status":"completed","output_text":"ok"}`), nil
		},
		HandleStreaming: func(context.Context, *http.Response, *ui.Spinner) (string, string, error) {
			return "", "", errors.New("unexpected streaming handler")
		},
		HandleNonStreaming: func(_ context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error) {
			api.StopSpinner(spinner)
			return HandleNonStreaming(api.WithAssistantUpdateMode(context.Background(), api.AssistantUpdatesOff), resp, nil, NonStreamingOptions{})
		},
		RequestObserver: func(req Request) {
			observed = append(observed, req)
		},
		HasPreviousResponseID: func() bool {
			return previousResponseID != ""
		},
		ClearPreviousResponseID: func() {
			previousResponseID = ""
		},
		ProviderName: "OpenAI",
	})
	if err != nil {
		t.Fatalf("RunResponsesRequest() error = %v", err)
	}
	if content != "ok" || responseID != "resp_new" {
		t.Fatalf("RunResponsesRequest() = (%q, %q), want ok and resp_new", content, responseID)
	}
	if executeCalls != 2 || len(payloads) != 2 || len(observed) != 2 {
		t.Fatalf("calls: execute=%d payloads=%d observed=%d, want 2 each", executeCalls, len(payloads), len(observed))
	}
	if !strings.Contains(payloads[0], `"previous_response_id":"resp_old"`) {
		t.Fatalf("first payload = %s, want previous_response_id", payloads[0])
	}
	if strings.Contains(payloads[1], "previous_response_id") {
		t.Fatalf("second payload = %s, want previous_response_id cleared", payloads[1])
	}
	if observed[0].PreviousResponseID != "resp_old" || observed[1].PreviousResponseID != "" {
		t.Fatalf("observed previous_response_id = [%q %q], want old then cleared", observed[0].PreviousResponseID, observed[1].PreviousResponseID)
	}
}

func newResponsesNonStreamingHTTPResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
