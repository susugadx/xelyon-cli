package api

import (
	"strings"
	"testing"
)

func TestValidateStreamResponse_Valid(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "with delta",
			json: `{"choices": [{"delta": {"content": "hello"}}]}`,
		},
		{
			name: "with message",
			json: `{"choices": [{"message": {"content": "hello"}}]}`,
		},
		{
			name: "multiple choices",
			json: `{"choices": [{"delta": {"content": "a"}}, {"delta": {"content": "b"}}]}`,
		},
		{
			name: "usage chunk with empty choices",
			json: `{"choices": [], "usage": {"prompt_tokens": 100, "completion_tokens": 50}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStreamResponse([]byte(tt.json))
			if err != nil {
				t.Errorf("ValidateStreamResponse failed: %v", err)
			}
		})
	}
}

func TestValidateStreamResponse_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantError string
	}{
		{
			name:      "invalid JSON",
			json:      `{invalid}`,
			wantError: "invalid JSON",
		},
		{
			name:      "missing choices",
			json:      `{"data": "test"}`,
			wantError: "missing required field: choices",
		},
		{
			name:      "empty choices array",
			json:      `{"choices": []}`,
			wantError: "choices must be a non-empty array",
		},
		{
			name:      "choices not array",
			json:      `{"choices": "not an array"}`,
			wantError: "choices must be an array",
		},
		{
			name:      "choice without delta or message",
			json:      `{"choices": [{"other": "field"}]}`,
			wantError: "must have 'delta' or 'message' field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStreamResponse([]byte(tt.json))
			if err == nil {
				t.Error("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("Expected error containing '%s', got: %v", tt.wantError, err)
			}
		})
	}
}

func TestValidateChatResponse_Valid(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "simple message",
			json: `{"choices": [{"message": {"content": "hello"}}]}`,
		},
		{
			name: "multiple choices",
			json: `{"choices": [{"message": {"content": "a"}}, {"message": {"content": "b"}}]}`,
		},
		{
			name: "with role",
			json: `{"choices": [{"message": {"role": "assistant", "content": "hello"}}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateChatResponse([]byte(tt.json))
			if err != nil {
				t.Errorf("ValidateChatResponse failed: %v", err)
			}
		})
	}
}

func TestValidateChatResponse_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantError string
	}{
		{
			name:      "invalid JSON",
			json:      `{bad json}`,
			wantError: "invalid JSON",
		},
		{
			name:      "missing choices",
			json:      `{"data": "test"}`,
			wantError: "missing required field: choices",
		},
		{
			name:      "empty choices",
			json:      `{"choices": []}`,
			wantError: "choices must be a non-empty array",
		},
		{
			name:      "message not object",
			json:      `{"choices": [{"message": "not an object"}]}`,
			wantError: "message must be an object",
		},
		{
			name:      "missing content",
			json:      `{"choices": [{"message": {"role": "assistant"}}]}`,
			wantError: "message.content is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateChatResponse([]byte(tt.json))
			if err == nil {
				t.Error("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("Expected error containing '%s', got: %v", tt.wantError, err)
			}
		})
	}
}

func TestValidateToolCall_Valid(t *testing.T) {
	tests := []struct {
		name     string
		toolCall map[string]interface{}
	}{
		{
			name: "simple tool call",
			toolCall: map[string]interface{}{
				"function": map[string]interface{}{
					"name":      "test_function",
					"arguments": "{}",
				},
			},
		},
		{
			name: "with id",
			toolCall: map[string]interface{}{
				"id":   "call_123",
				"type": "function",
				"function": map[string]interface{}{
					"name":      "test_function",
					"arguments": `{"arg": "value"}`,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToolCall(tt.toolCall)
			if err != nil {
				t.Errorf("ValidateToolCall failed: %v", err)
			}
		})
	}
}

func TestValidateToolCall_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		toolCall  interface{}
		wantError string
	}{
		{
			name:      "not an object",
			toolCall:  "not an object",
			wantError: "tool_call must be an object",
		},
		{
			name: "missing function",
			toolCall: map[string]interface{}{
				"id": "call_123",
			},
			wantError: "function must be an object",
		},
		{
			name: "missing name",
			toolCall: map[string]interface{}{
				"function": map[string]interface{}{
					"arguments": "{}",
				},
			},
			wantError: "function.name is required",
		},
		{
			name: "missing arguments",
			toolCall: map[string]interface{}{
				"function": map[string]interface{}{
					"name": "test_function",
				},
			},
			wantError: "function.arguments is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToolCall(tt.toolCall)
			if err == nil {
				t.Error("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("Expected error containing '%s', got: %v", tt.wantError, err)
			}
		})
	}
}

func TestValidateErrorResponse(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		wantMessage string
		wantErr     bool
	}{
		{
			name:        "with error.message",
			json:        `{"error": {"message": "API key is invalid"}}`,
			wantMessage: "API key is invalid",
			wantErr:     false,
		},
		{
			name:        "error as string",
			json:        `{"error": "Something went wrong"}`,
			wantMessage: "Something went wrong",
			wantErr:     false,
		},
		{
			name:        "no error field",
			json:        `{"status": "failed", "reason": "unknown"}`,
			wantMessage: `{"status": "failed", "reason": "unknown"}`,
			wantErr:     false,
		},
		{
			name:    "invalid JSON",
			json:    `{bad json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ValidateErrorResponse([]byte(tt.json))

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !strings.Contains(msg, tt.wantMessage) {
				t.Errorf("Expected message containing '%s', got: %s", tt.wantMessage, msg)
			}
		})
	}
}
