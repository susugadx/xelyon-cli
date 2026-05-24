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

func TestBuildChatMessages_StandardMessagesPayloadUnchanged(t *testing.T) {
	messages := BuildChatMessages("system", []api.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	})

	payload := marshalMessagesPayload(t, messages)
	if len(payload) != 3 {
		t.Fatalf("len(messages) = %d, want 3", len(payload))
	}
	assertPayloadKeys(t, payload[0], "role", "content")
	assertPayloadKeys(t, payload[1], "role", "content")
	assertPayloadKeys(t, payload[2], "role", "content")
	if payload[0]["role"] != "system" || payload[0]["content"] != "system" {
		t.Fatalf("system payload = %#v, want role/content only", payload[0])
	}
	if payload[1]["role"] != "user" || payload[1]["content"] != "hello" {
		t.Fatalf("user payload = %#v, want role/content only", payload[1])
	}
	if payload[2]["role"] != "assistant" || payload[2]["content"] != "hi" {
		t.Fatalf("assistant payload = %#v, want role/content only", payload[2])
	}
}

func TestBuildChatMessagesWithActiveContext_InsertsEphemeralSystemBeforeHistory(t *testing.T) {
	messages := BuildChatMessagesWithActiveContext("system", []api.ActiveContextBlock{
		{Name: "blank", Content: "\n \n"},
		{Name: "state", Content: "<current_task_state>\nstate\n</current_task_state>"},
		{Name: "evidence", Content: "<rehydrated_evidence>\nevidence\n</rehydrated_evidence>"},
	}, []api.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	})

	if len(messages) != 4 {
		t.Fatalf("len(messages) = %d, want system + active context + history", len(messages))
	}
	if messages[0].Role != "system" || messages[0].Content != "system" {
		t.Fatalf("messages[0] = %#v, want base system prompt", messages[0])
	}
	if messages[1].Role != "system" {
		t.Fatalf("messages[1].Role = %q, want system active context", messages[1].Role)
	}
	wantActive := "<current_task_state>\nstate\n</current_task_state>\n\n<rehydrated_evidence>\nevidence\n</rehydrated_evidence>"
	if messages[1].Content != wantActive {
		t.Fatalf("messages[1].Content = %q, want rendered active context", messages[1].Content)
	}
	if messages[2].Role != "user" || messages[2].Content != "hello" {
		t.Fatalf("messages[2] = %#v, want first history message after active context", messages[2])
	}
}

func TestBuildChatMessageInterfacesWithActiveContext_UsesSharedEphemeralSystemPlacement(t *testing.T) {
	messages := BuildChatMessageInterfacesWithActiveContext("system", []api.ActiveContextBlock{{
		Name:    "state",
		Content: "<current_task_state>\nstate\n</current_task_state>",
	}}, []api.Message{{
		Role:     "tool",
		ToolName: "read_file",
		Content:  "result",
	}}, func(message api.Message) api.Message {
		message.ToolName = ""
		return message
	})

	if len(messages) != 3 {
		t.Fatalf("len(messages) = %d, want system + active context + history", len(messages))
	}
	active, ok := messages[1].(api.Message)
	if !ok || active.Role != "system" || active.Content != "<current_task_state>\nstate\n</current_task_state>" {
		t.Fatalf("messages[1] = %#v, want active context system message", messages[1])
	}
	history, ok := messages[2].(api.Message)
	if !ok || history.Role != "tool" || history.ToolName != "" || history.Content != "result" {
		t.Fatalf("messages[2] = %#v, want transformed history message", messages[2])
	}
}

func TestBuildChatMessages_OmitsEmptyReasoningContent(t *testing.T) {
	messages := BuildChatMessages("system", []api.Message{
		{Role: "assistant", Content: "hi", ReasoningContent: ""},
	})

	payload := marshalMessagesPayload(t, messages)
	if _, ok := payload[1]["reasoning_content"]; ok {
		t.Fatalf("reasoning_content = %#v, want omitted when empty", payload[1]["reasoning_content"])
	}
}

func TestBuildChatMessages_PreservesReasoningAndToolFields(t *testing.T) {
	messages := BuildChatMessages("system", []api.Message{
		{
			Role:             "assistant",
			Content:          "I'll inspect it.",
			ReasoningContent: "Need to inspect the file first.",
			ToolCalls: []api.OpenAIToolCall{{
				ID:   "call_1",
				Type: "function",
				Function: api.OpenAIToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"README.md"}`,
				},
			}},
		},
		{Role: "tool", ToolCallID: "call_1", Content: "README contents"},
	})

	payload := marshalMessagesPayload(t, messages)
	assistant := payload[1]
	if assistant["reasoning_content"] != "Need to inspect the file first." {
		t.Fatalf("assistant reasoning_content = %#v, want preserved", assistant["reasoning_content"])
	}
	toolCalls, ok := assistant["tool_calls"].([]any)
	if !ok || len(toolCalls) != 1 {
		t.Fatalf("assistant tool_calls = %#v, want one tool call", assistant["tool_calls"])
	}
	toolCall, ok := toolCalls[0].(map[string]any)
	if !ok || toolCall["id"] != "call_1" || toolCall["type"] != "function" {
		t.Fatalf("tool_call = %#v, want id/type preserved", toolCalls[0])
	}
	function, ok := toolCall["function"].(map[string]any)
	if !ok || function["name"] != "read_file" || function["arguments"] != `{"path":"README.md"}` {
		t.Fatalf("tool_call function = %#v, want read_file arguments preserved", toolCall["function"])
	}

	tool := payload[2]
	if tool["role"] != "tool" || tool["tool_call_id"] != "call_1" || tool["content"] != "README contents" {
		t.Fatalf("tool payload = %#v, want role/tool_call_id/content preserved", tool)
	}
}

func TestBuildChatCompletionsRequest_DefaultMaxTokensField(t *testing.T) {
	req := BuildChatCompletionsRequest(ChatCompletionsRequestOptions{
		Model:     "provider/model",
		Messages:  []api.Message{{Role: "user", Content: "hello"}},
		MaxTokens: 123,
	})

	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body["max_tokens"] != float64(123) {
		t.Fatalf("max_tokens = %v, want 123", body["max_tokens"])
	}
	if _, ok := body["max_completion_tokens"]; ok {
		t.Fatalf("max_completion_tokens = %v, want absent", body["max_completion_tokens"])
	}
}

func TestBuildChatCompletionsRequest_MaxCompletionTokensField(t *testing.T) {
	req := BuildChatCompletionsRequest(ChatCompletionsRequestOptions{
		Model:               "provider/model",
		Messages:            []api.Message{{Role: "user", Content: "hello"}},
		MaxTokens:           123,
		MaxCompletionTokens: 456,
	})

	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body["max_completion_tokens"] != float64(456) {
		t.Fatalf("max_completion_tokens = %v, want 456", body["max_completion_tokens"])
	}
	if _, ok := body["max_tokens"]; ok {
		t.Fatalf("max_tokens = %v, want absent", body["max_tokens"])
	}
}

func TestChatCompletionsRequest_RejectsExtraFieldConflict(t *testing.T) {
	tests := []struct {
		name       string
		extraField string
	}{
		{name: "model", extraField: "model"},
		{name: "max tokens", extraField: "max_tokens"},
		{name: "max completion tokens", extraField: "max_completion_tokens"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ChatCompletionsRequest{
				Model:       "provider/model",
				Messages:    []api.Message{{Role: "user", Content: "hello"}},
				ExtraFields: map[string]any{tt.extraField: "override"},
			}

			if _, err := json.Marshal(req); err == nil {
				t.Fatal("json.Marshal() error = nil, want conflict error")
			}
		})
	}
}

func TestToolChoicePolicies(t *testing.T) {
	toolName := "read_file"

	defaultChoice := DefaultToolChoicePolicy(&toolName)
	defaultBody, ok := defaultChoice.(map[string]interface{})
	if !ok || defaultBody["type"] != "function" {
		t.Fatalf("DefaultToolChoicePolicy() = %#v, want forced function", defaultChoice)
	}

	if got := DefaultToolChoicePolicy(nil); got != "auto" {
		t.Fatalf("DefaultToolChoicePolicy(nil) = %v, want auto", got)
	}
	if got := AutoToolChoicePolicy(&toolName); got != "auto" {
		t.Fatalf("AutoToolChoicePolicy(toolName) = %v, want auto", got)
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

func marshalMessagesPayload(t *testing.T, messages []api.Message) []map[string]any {
	t.Helper()
	payload, err := json.Marshal(messages)
	if err != nil {
		t.Fatalf("json.Marshal(messages) error = %v", err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(messages) error = %v", err)
	}
	return decoded
}

func assertPayloadKeys(t *testing.T, payload map[string]any, keys ...string) {
	t.Helper()
	if len(payload) != len(keys) {
		t.Fatalf("payload keys = %#v, want only %v", payload, keys)
	}
	for _, key := range keys {
		if _, ok := payload[key]; !ok {
			t.Fatalf("payload keys = %#v, missing %q", payload, key)
		}
	}
}
