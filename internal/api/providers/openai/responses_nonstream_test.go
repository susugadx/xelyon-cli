package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestHandleResponsesNonStreaming_MessageOutputTextUsageAndResponseID(t *testing.T) {
	p := New("test-key")
	var gotUsage api.Usage
	p.SetUsageCallback(func(usage api.Usage) {
		gotUsage = usage
	})

	resp := newResponsesNonStreamingHTTPResponse(`{
		"id": "resp_123",
		"status": "completed",
		"output": [{
			"type": "message",
			"content": [
				{"type": "output_text", "text": "Hello "},
				{"type": "output_text", "text": "world"}
			]
		}],
		"usage": {
			"input_tokens": 10,
			"output_tokens": 4,
			"input_tokens_details": {"cached_tokens": 3},
			"output_tokens_details": {"reasoning_tokens": 2}
		}
	}`)
	defer resp.Body.Close()

	content, responseID, err := p.handleResponsesNonStreaming(newOpenAITestContext(t, false), resp, nil)
	if err != nil {
		t.Fatalf("handleResponsesNonStreaming() error = %v", err)
	}
	if content != "Hello world" {
		t.Fatalf("content = %q, want Hello world", content)
	}
	if responseID != "resp_123" {
		t.Fatalf("responseID = %q, want resp_123", responseID)
	}
	if gotUsage.InputTokens != 10 || gotUsage.OutputTokens != 2 || gotUsage.CachedInputTokens != 3 || gotUsage.ThinkingTokens != 2 {
		t.Fatalf("usage = %+v, want input=10 output=2 cached=3 thinking=2", gotUsage)
	}
}

func TestHandleResponsesNonStreaming_TextAndFunctionCall(t *testing.T) {
	p := New("test-key")
	resp := newResponsesNonStreamingHTTPResponse(`{
		"id": "resp_tool",
		"output_text": "Need a file",
		"output": [{
			"type": "function_call",
			"call_id": "call_1",
			"name": "read_file",
			"arguments": "{\"paths\":[\"main.go\"]}"
		}]
	}`)
	defer resp.Body.Close()

	content, responseID, err := p.handleResponsesNonStreaming(newOpenAITestContext(t, false), resp, nil)
	if err != nil {
		t.Fatalf("handleResponsesNonStreaming() error = %v", err)
	}
	if responseID != "resp_tool" {
		t.Fatalf("responseID = %q, want resp_tool", responseID)
	}
	if !strings.HasPrefix(content, "Need a file") {
		t.Fatalf("content = %q, want text prefix", content)
	}
	if !strings.Contains(content, `"id":"call_1"`) ||
		!strings.Contains(content, `"tool":"read_file"`) ||
		!strings.Contains(content, `"paths":["main.go"]`) {
		t.Fatalf("content = %q, want internal tool JSON", content)
	}
}

func TestHandleResponsesNonStreaming_ErrorStatusesDoNotReturnContentOrUsage(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantErrSub string
	}{
		{
			name: "failed with error message",
			body: `{
				"id": "resp_failed",
				"status": "failed",
				"error": {"message": "model failed"},
				"output_text": "ignored",
				"usage": {"input_tokens": 10, "output_tokens": 2}
			}`,
			wantErrSub: "model failed",
		},
		{
			name:       "failed without error",
			body:       `{"id": "resp_failed", "status": "failed", "output_text": "ignored"}`,
			wantErrSub: "OpenAI Responses API request failed",
		},
		{
			name:       "non completed status",
			body:       `{"id": "resp_incomplete", "status": "incomplete", "output_text": "partial"}`,
			wantErrSub: "OpenAI Responses API response status: incomplete",
		},
		{
			name:       "error code without message",
			body:       `{"id": "resp_error", "status": "completed", "error": {"code": "rate_limit_exceeded"}}`,
			wantErrSub: "OpenAI API error: rate_limit_exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New("test-key")
			usageCalls := 0
			p.SetUsageCallback(func(usage api.Usage) {
				usageCalls++
			})
			resp := newResponsesNonStreamingHTTPResponse(tt.body)
			defer resp.Body.Close()

			content, responseID, err := p.handleResponsesNonStreaming(newOpenAITestContext(t, false), resp, nil)
			if err == nil {
				t.Fatal("handleResponsesNonStreaming() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErrSub)
			}
			if content != "" {
				t.Fatalf("content = %q, want empty", content)
			}
			if responseID != "" {
				t.Fatalf("responseID = %q, want empty", responseID)
			}
			if usageCalls != 0 {
				t.Fatalf("usage calls = %d, want 0", usageCalls)
			}
		})
	}
}

func TestChatWithResponses_GPT55ProUsesNonStreamingAndStoresResponseID(t *testing.T) {
	var raw map[string]any
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if raw["stream"] == true {
			t.Fatalf("stream = true, want false or omitted for gpt-5.5-pro")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_pro","output_text":"Pro response"}`))
	})
	t.Setenv("OPENAI_RESPONSES_URL", server.URL)

	p := New("test-key")
	content, err := p.chatWithResponses(newOpenAITestContext(t, false), "system", []api.Message{{Role: "user", Content: "hi"}}, "gpt-5.5-pro")
	if err != nil {
		t.Fatalf("chatWithResponses() error = %v", err)
	}
	if content != "Pro response" {
		t.Fatalf("content = %q, want Pro response", content)
	}
	if p.GetResponseID() != "resp_pro" {
		t.Fatalf("GetResponseID() = %q, want resp_pro", p.GetResponseID())
	}
	if raw["model"] != "gpt-5.5-pro" {
		t.Fatalf("model = %#v, want gpt-5.5-pro", raw["model"])
	}
}

func TestChatWithResponses_NonStreamingFailedStatusDoesNotStoreResponseID(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "resp_failed",
			"status": "failed",
			"error": {"message": "model failed"}
		}`))
	})
	t.Setenv("OPENAI_RESPONSES_URL", server.URL)

	p := New("test-key")
	content, err := p.chatWithResponses(newOpenAITestContext(t, false), "system", []api.Message{{Role: "user", Content: "hi"}}, "gpt-5.5-pro")
	if err == nil {
		t.Fatal("chatWithResponses() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "model failed") {
		t.Fatalf("error = %q, want model failed", err.Error())
	}
	if content != "" {
		t.Fatalf("content = %q, want empty", content)
	}
	if p.GetResponseID() != "" {
		t.Fatalf("GetResponseID() = %q, want empty", p.GetResponseID())
	}
}

func TestChatWithResponses_NonStreamingRetriesWithoutInvalidPreviousResponseID(t *testing.T) {
	var requests []map[string]any
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		var raw map[string]any
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		requests = append(requests, raw)
		if raw["stream"] == true {
			t.Fatalf("stream = true, want false or omitted for gpt-5.5-pro")
		}

		if len(requests) == 1 {
			if raw["previous_response_id"] != "resp_old" {
				t.Fatalf("first previous_response_id = %#v, want resp_old", raw["previous_response_id"])
			}
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"invalid previous_response_id"}}`))
			return
		}

		if _, ok := raw["previous_response_id"]; ok {
			t.Fatalf("second request should clear previous_response_id: %#v", raw)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_new","output_text":"Recovered"}`))
	})
	t.Setenv("OPENAI_RESPONSES_URL", server.URL)

	p := New("test-key")
	p.SetResponseID("resp_old")
	content, err := p.chatWithResponses(newOpenAITestContext(t, false), "system", []api.Message{{Role: "user", Content: "hi"}}, "gpt-5.5-pro")
	if err != nil {
		t.Fatalf("chatWithResponses() error = %v", err)
	}
	if content != "Recovered" {
		t.Fatalf("content = %q, want Recovered", content)
	}
	if p.GetResponseID() != "resp_new" {
		t.Fatalf("GetResponseID() = %q, want resp_new", p.GetResponseID())
	}
	if len(requests) != 2 {
		t.Fatalf("request count = %d, want 2", len(requests))
	}
}

func newResponsesNonStreamingHTTPResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestSupportsResponsesStreaming(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{model: "gpt-5.5", want: true},
		{model: "gpt-5.5-2026-04-23", want: true},
		{model: "gpt-5.5-pro", want: false},
		{model: "gpt-5.5-pro-2026-04-23", want: false},
		{model: "gpt-5.4-pro", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := supportsResponsesStreaming(tt.model); got != tt.want {
				t.Fatalf("supportsResponsesStreaming(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestBuildChatResponsesRequest_UsesCatalogModelForGPT55ProStreamingGate(t *testing.T) {
	cfg := configWithGPT55ProDeployment()
	ctx := config.WithContext(context.Background(), cfg)

	req := New("test-key").buildChatResponsesRequest(ctx, "system", []api.Message{{Role: "user", Content: "hi"}}, "corp-pro-deployment")
	if req.Stream {
		t.Fatal("Stream = true, want false via catalog_model gpt-5.5-pro")
	}
}

func configWithGPT55ProDeployment() *config.Config {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		DefaultModel: "corp-pro-deployment",
		CatalogModel: "gpt-5.5-pro",
	})
	return cfg
}
