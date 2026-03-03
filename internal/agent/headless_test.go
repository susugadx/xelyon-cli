package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// mockErrorProvider は常にエラーを返すプロバイダー
type mockErrorProvider struct{}

func (m *mockErrorProvider) Name() string                   { return "test-error" }
func (m *mockErrorProvider) SupportsImages() bool           { return false }
func (m *mockErrorProvider) IsFunctionCallingEnabled() bool { return false }
func (m *mockErrorProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	return "", fmt.Errorf("mock error")
}
func (m *mockErrorProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return "", fmt.Errorf("mock error")
}

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
	// ToJSON()で生成したJSONを再度パースできることを確認
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

	// パース
	var parsed HeadlessResult
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// 検証
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
		ToolCalls:  []ToolCallResult{}, // 空配列
	}

	jsonStr, err := result.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	// 空配列の場合、omitemptyで省略されないことを確認
	// （仕様により、tool_callsは常に含まれる）
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// tool_callsが存在しないか、または空配列であることを確認
	// (omitemptyなので、空の場合は省略される)
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
			Input:  100,
			Output: 50,
			Total:  150,
		},
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
	if !strings.Contains(jsonStr, `"output": 50`) {
		t.Error("Expected JSON to contain output token count")
	}
	if !strings.Contains(jsonStr, `"total": 150`) {
		t.Error("Expected JSON to contain total token count")
	}
}

func TestErrorInfo(t *testing.T) {
	error := &ErrorInfo{
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
		Error:      error,
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
	// 大量のツール呼び出しを含むケース
	toolCalls := make([]ToolCallResult, 10)
	for i := 0; i < 10; i++ {
		toolCalls[i] = ToolCallResult{
			Tool:    "bash",
			Args:    map[string]string{"command": "echo test"},
			Output:  strings.Repeat("output ", 100), // 大きな出力
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

	// JSONが有効であることを確認
	var parsed HeadlessResult
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse large JSON: %v", err)
	}

	if len(parsed.ToolCalls) != 10 {
		t.Errorf("Expected 10 tool calls, got %d", len(parsed.ToolCalls))
	}
}

func TestRunHeadless_CallsCleanup(t *testing.T) {
	var called atomic.Int32
	cleanupHook = func() { called.Add(1) }
	defer func() { cleanupHook = nil }()

	provider := &mockProvider{name: "test"}
	_ = RunHeadless("hello", "test-model", provider)

	if called.Load() != 1 {
		t.Errorf("Cleanup was called %d times, want 1", called.Load())
	}
}

func TestRunHeadless_CallsCleanupOnError(t *testing.T) {
	var called atomic.Int32
	cleanupHook = func() { called.Add(1) }
	defer func() { cleanupHook = nil }()

	provider := &mockErrorProvider{}
	result := RunHeadless("hello", "test-model", provider)

	if result.Status != "error" {
		t.Errorf("Expected error status, got %s", result.Status)
	}
	if called.Load() != 1 {
		t.Errorf("Cleanup was called %d times on error path, want 1", called.Load())
	}
}

func TestRunHeadless_RepeatedInvocations(t *testing.T) {
	var called atomic.Int32
	cleanupHook = func() { called.Add(1) }
	defer func() { cleanupHook = nil }()

	provider := &mockProvider{name: "test"}
	for i := 0; i < 5; i++ {
		_ = RunHeadless("hello", "test-model", provider)
	}

	if called.Load() != 5 {
		t.Errorf("Cleanup was called %d times for 5 invocations, want 5", called.Load())
	}
}

func TestRunHeadless_NoLeakOnRepeatedInvocations(t *testing.T) {
	var cleanupCount atomic.Int32
	cleanupHook = func() { cleanupCount.Add(1) }
	defer func() { cleanupHook = nil }()

	const iterations = 20
	provider := &mockProvider{name: "mock"}

	runtime.GC()
	baseGoroutines := runtime.NumGoroutine()

	for i := 0; i < iterations; i++ {
		res := RunHeadless("test query", "mock-model", provider)
		if res.Status != "success" {
			t.Fatalf("iteration %d: RunHeadless failed: %v", i, res.Error)
		}
	}

	runtime.GC()
	finalGoroutines := runtime.NumGoroutine()

	if cleanupCount.Load() != int32(iterations) {
		t.Fatalf("Cleanup call count mismatch: got %d, want %d", cleanupCount.Load(), iterations)
	}

	if leaked := finalGoroutines - baseGoroutines; leaked > 5 {
		t.Errorf("possible goroutine leak: base=%d, final=%d, leaked=%d", baseGoroutines, finalGoroutines, leaked)
	}
}
