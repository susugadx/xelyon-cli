package agent

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestHeadlessResult_ToJSON(t *testing.T) {
	result := &HeadlessResult{
		Status:     "success",
		Provider:   "DeepSeek",
		Model:      "deepseek-coder",
		Response:   "Test response",
		DurationMs: 1500,
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	jsonStr, err := result.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	if !strings.Contains(jsonStr, `"status": "success"`) {
		t.Error("Expected JSON to contain status field")
	}
	if !strings.Contains(jsonStr, `"schema_version": "xelyon.headless.v1"`) {
		t.Error("Expected JSON to contain schema_version field")
	}
	if !strings.Contains(jsonStr, `"provider": "DeepSeek"`) {
		t.Error("Expected JSON to contain provider field")
	}
	if !strings.Contains(jsonStr, `"model": "deepseek-coder"`) {
		t.Error("Expected JSON to contain model field")
	}
	if !strings.Contains(jsonStr, `"response": "Test response"`) {
		t.Error("Expected JSON to contain response field")
	}
}

func TestHeadlessResult_ConstructorsSetSchemaVersion(t *testing.T) {
	tests := []struct {
		name   string
		result *HeadlessResult
	}{
		{
			name:   "success",
			result: NewSuccessResult("openai", "gpt-5.4", "ok", nil, 10),
		},
		{
			name:   "error",
			result: NewErrorResult("openai", "gpt-5.4", HeadlessErrorTypeConfig, "bad input", 10),
		},
		{
			name:   "tool loop limit",
			result: NewToolLoopLimitResult("openai", "gpt-5.4", 3, nil, 10),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.SchemaVersion != HeadlessSchemaVersion {
				t.Fatalf("SchemaVersion = %q, want %q", tt.result.SchemaVersion, HeadlessSchemaVersion)
			}
		})
	}
}

func TestHeadlessResult_ToJSON_WithToolCalls(t *testing.T) {
	result := &HeadlessResult{
		Status:   "success",
		Provider: "DeepSeek",
		Model:    "deepseek-coder",
		Response: "File read successfully",
		ToolCalls: []ToolCallResult{
			{
				Tool:    "read_file",
				Args:    map[string]string{"path": "main.go"},
				Output:  "package main...",
				Success: true,
			},
			{
				Tool:    "bash",
				Args:    map[string]string{"command": "go test"},
				Output:  "PASS",
				Success: true,
			},
		},
		DurationMs: 2000,
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	jsonStr, err := result.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	if !strings.Contains(jsonStr, `"tool_calls"`) {
		t.Error("Expected JSON to contain tool_calls field")
	}
	if !strings.Contains(jsonStr, `"read_file"`) {
		t.Error("Expected JSON to contain read_file tool")
	}
	if !strings.Contains(jsonStr, `"bash"`) {
		t.Error("Expected JSON to contain bash tool")
	}
}

func TestHeadlessResult_ToJSON_WithError(t *testing.T) {
	result := &HeadlessResult{
		Status:     "error",
		Provider:   "DeepSeek",
		Model:      "deepseek-coder",
		Response:   "",
		DurationMs: 500,
		Timestamp:  time.Now().Format(time.RFC3339),
		Error: &ErrorInfo{
			Type:    "api_error",
			Message: "API key not set",
			Code:    401,
		},
	}

	jsonStr, err := result.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	if !strings.Contains(jsonStr, `"status": "error"`) {
		t.Error("Expected JSON to contain error status")
	}
	if !strings.Contains(jsonStr, `"error"`) {
		t.Error("Expected JSON to contain error field")
	}
	if !strings.Contains(jsonStr, `"api_error"`) {
		t.Error("Expected JSON to contain error type")
	}
	if !strings.Contains(jsonStr, `"API key not set"`) {
		t.Error("Expected JSON to contain error message")
	}
}

func TestHeadlessResult_JSONUnmarshal(t *testing.T) {
	original := &HeadlessResult{
		Status:     "success",
		Provider:   "Claude",
		Model:      "claude-3-opus",
		Response:   "Analysis complete",
		DurationMs: 3000,
		Timestamp:  "2026-01-11T10:00:00Z",
		ToolCalls: []ToolCallResult{
			{
				Tool:    "read_file",
				Args:    map[string]string{"path": "test.go"},
				Output:  "content",
				Success: true,
			},
		},
	}

	jsonStr, err := original.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	var parsed HeadlessResult
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed.Status != original.Status {
		t.Errorf("Expected status '%s', got '%s'", original.Status, parsed.Status)
	}
	if parsed.Provider != original.Provider {
		t.Errorf("Expected provider '%s', got '%s'", original.Provider, parsed.Provider)
	}
	if parsed.Model != original.Model {
		t.Errorf("Expected model '%s', got '%s'", original.Model, parsed.Model)
	}
	if parsed.Response != original.Response {
		t.Errorf("Expected response '%s', got '%s'", original.Response, parsed.Response)
	}
	if parsed.DurationMs != original.DurationMs {
		t.Errorf("Expected duration %d, got %d", original.DurationMs, parsed.DurationMs)
	}
	if len(parsed.ToolCalls) != len(original.ToolCalls) {
		t.Errorf("Expected %d tool calls, got %d", len(original.ToolCalls), len(parsed.ToolCalls))
	}
}

func TestHeadlessResult_EmptyToolCalls(t *testing.T) {
	result := &HeadlessResult{
		Status:     "success",
		Provider:   "DeepSeek",
		Model:      "deepseek-coder",
		Response:   "Simple response without tools",
		DurationMs: 800,
		Timestamp:  time.Now().Format(time.RFC3339),
		ToolCalls:  []ToolCallResult{},
	}

	jsonStr, err := result.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if toolCalls, ok := parsed["tool_calls"]; ok {
		if arr, ok := toolCalls.([]interface{}); ok {
			if len(arr) != 0 {
				t.Errorf("Expected empty tool_calls array, got %d items", len(arr))
			}
		}
	}
}

func TestTokenUsage(t *testing.T) {
	result := &HeadlessResult{
		Status:     "success",
		Provider:   "OpenAI",
		Model:      "gpt-4o",
		Response:   "Response with token tracking",
		DurationMs: 2000,
		Timestamp:  time.Now().Format(time.RFC3339),
		Tokens: &TokenUsage{
			Input:    100,
			Cached:   25,
			Output:   50,
			Thinking: 10,
			Total:    160,
		},
		Cost: 0.1234,
	}

	jsonStr, err := result.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	if !strings.Contains(jsonStr, `"tokens"`) {
		t.Error("Expected JSON to contain tokens field")
	}
	if !strings.Contains(jsonStr, `"input": 100`) {
		t.Error("Expected JSON to contain input token count")
	}
	if !strings.Contains(jsonStr, `"cached": 25`) {
		t.Error("Expected JSON to contain cached token count")
	}
	if !strings.Contains(jsonStr, `"output": 50`) {
		t.Error("Expected JSON to contain output token count")
	}
	if !strings.Contains(jsonStr, `"thinking": 10`) {
		t.Error("Expected JSON to contain thinking token count")
	}
	if !strings.Contains(jsonStr, `"total": 160`) {
		t.Error("Expected JSON to contain total token count")
	}
	if !strings.Contains(jsonStr, `"cost": 0.1234`) {
		t.Error("Expected JSON to contain cost")
	}
}

func TestErrorInfo(t *testing.T) {
	errorInfo := &ErrorInfo{
		Type:    "tool_error",
		Message: "File not found",
		Code:    404,
	}

	result := &HeadlessResult{
		Status:     "error",
		Provider:   "DeepSeek",
		Model:      "deepseek-coder",
		Response:   "",
		DurationMs: 100,
		Timestamp:  time.Now().Format(time.RFC3339),
		Error:      errorInfo,
	}

	jsonStr, err := result.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	var parsed HeadlessResult
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed.Error == nil {
		t.Fatal("Expected error field to be present")
	}
	if parsed.Error.Type != "tool_error" {
		t.Errorf("Expected error type 'tool_error', got '%s'", parsed.Error.Type)
	}
	if parsed.Error.Message != "File not found" {
		t.Errorf("Expected error message 'File not found', got '%s'", parsed.Error.Message)
	}
	if parsed.Error.Code != 404 {
		t.Errorf("Expected error code 404, got %d", parsed.Error.Code)
	}
}

func TestHeadlessResult_LargeOutput(t *testing.T) {
	toolCalls := make([]ToolCallResult, 10)
	for i := 0; i < 10; i++ {
		toolCalls[i] = ToolCallResult{
			Tool:    "bash",
			Args:    map[string]string{"command": "echo test"},
			Output:  strings.Repeat("output ", 100),
			Success: true,
		}
	}

	result := &HeadlessResult{
		Status:     "success",
		Provider:   "DeepSeek",
		Model:      "deepseek-coder",
		Response:   strings.Repeat("Long response ", 100),
		ToolCalls:  toolCalls,
		DurationMs: 5000,
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	jsonStr, err := result.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert large result to JSON: %v", err)
	}

	var parsed HeadlessResult
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse large JSON: %v", err)
	}

	if len(parsed.ToolCalls) != 10 {
		t.Errorf("Expected 10 tool calls, got %d", len(parsed.ToolCalls))
	}
}
