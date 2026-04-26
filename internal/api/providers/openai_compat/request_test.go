package openaicompat

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestBuildChatCompletionsRequest_BuildsStandardPayloadWithExtras(t *testing.T) {
	toolName := "read_file"
	extraFields := map[string]any{
		"thinking": map[string]any{"type": "disabled"},
	}

	req := BuildChatCompletionsRequest(ChatCompletionsRequestOptions{
		Model:        "provider/model",
		SystemPrompt: "system",
		History:      []api.Message{{Role: "user", Content: "hello"}},
		MaxTokens:    123,
		Stream:       true,
		IncludeUsage: true,
		FunctionCalling: &FunctionCallingOptions{
			Tools: []api.OpenAITool{{
				Type:     "function",
				Function: &api.ToolDefinition{Name: "read_file"},
			}},
			ToolName: &toolName,
		},
		ExtraFields: extraFields,
	})
	extraFields["thinking"] = "mutated"

	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body["model"] != "provider/model" {
		t.Fatalf("model = %v, want provider/model", body["model"])
	}
	messages, ok := body["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %v, want system + history", body["messages"])
	}
	streamOptions, ok := body["stream_options"].(map[string]any)
	if !ok || streamOptions["include_usage"] != true {
		t.Fatalf("stream_options = %v, want include_usage=true", body["stream_options"])
	}
	toolChoice, ok := body["tool_choice"].(map[string]any)
	if !ok || toolChoice["type"] != "function" {
		t.Fatalf("tool_choice = %v, want forced function", body["tool_choice"])
	}
	thinking, ok := body["thinking"].(map[string]any)
	if !ok || thinking["type"] != "disabled" {
		t.Fatalf("thinking = %v, want cloned extra field", body["thinking"])
	}
}

func TestChatCompletionsRequest_RejectsExtraFieldConflict(t *testing.T) {
	req := ChatCompletionsRequest{
		Model:       "provider/model",
		Messages:    []api.Message{{Role: "user", Content: "hello"}},
		ExtraFields: map[string]any{"model": "override"},
	}

	if _, err := json.Marshal(req); err == nil {
		t.Fatal("json.Marshal() error = nil, want conflict error")
	}
}

func TestJSONRequestAuthHelpers(t *testing.T) {
	body := ChatCompletionsRequest{
		Model:    "deployment",
		Messages: []api.Message{{Role: "user", Content: "hello"}},
	}

	bearerReq, err := NewBearerJSONRequest(context.Background(), "https://example.test/chat", "sk-test", body)
	if err != nil {
		t.Fatalf("NewBearerJSONRequest() error = %v", err)
	}
	if got := bearerReq.Header.Get("Authorization"); got != "Bearer sk-test" {
		t.Fatalf("Authorization = %q, want Bearer token", got)
	}
	if got := bearerReq.Header.Get("api-key"); got != "" {
		t.Fatalf("api-key = %q, want empty for bearer request", got)
	}

	apiKeyReq, err := NewAPIKeyJSONRequest(context.Background(), "https://example.test/chat", "azure-key", body)
	if err != nil {
		t.Fatalf("NewAPIKeyJSONRequest() error = %v", err)
	}
	if got := apiKeyReq.Header.Get("api-key"); got != "azure-key" {
		t.Fatalf("api-key = %q, want Azure key", got)
	}
	if got := apiKeyReq.Header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization = %q, want empty for api-key request", got)
	}
	assertJSONPostRequest(t, apiKeyReq)
}

func assertJSONPostRequest(t *testing.T, req *http.Request) {
	t.Helper()
	if req.Method != http.MethodPost {
		t.Fatalf("Method = %q, want POST", req.Method)
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	payload, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("ReadAll(body) error = %v", err)
	}
	var decoded ChatCompletionsRequest
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("request body is not ChatCompletionsRequest JSON: %v", err)
	}
	if decoded.Model != "deployment" {
		t.Fatalf("decoded model = %q, want deployment", decoded.Model)
	}
}
