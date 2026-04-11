package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
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

type headlessToolSetProbeProvider struct {
	name         string
	systemPrompt string
	toolNames    []string
}

func (p *headlessToolSetProbeProvider) Name() string {
	if p.name != "" {
		return p.name
	}
	return "openai"
}

func (p *headlessToolSetProbeProvider) SupportsImages() bool { return false }

func (p *headlessToolSetProbeProvider) IsFunctionCallingEnabled() bool { return true }

func (p *headlessToolSetProbeProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	p.systemPrompt = systemPrompt
	defs := tools.RegistryFromContext(ctx).GetToolDefinitions()
	p.toolNames = make([]string, len(defs))
	for i, def := range defs {
		p.toolNames[i] = def.Name
	}
	return "done", nil
}

func (p *headlessToolSetProbeProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

type headlessUsageProvider struct {
	usageCallback api.UsageCallback
}

func (p *headlessUsageProvider) Name() string { return "openai" }

func (p *headlessUsageProvider) SupportsImages() bool { return false }

func (p *headlessUsageProvider) IsFunctionCallingEnabled() bool { return true }

func (p *headlessUsageProvider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}

func (p *headlessUsageProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	if p.usageCallback != nil {
		p.usageCallback(api.Usage{
			InputTokens:       1000,
			CachedInputTokens: 200,
			OutputTokens:      300,
			ThinkingTokens:    50,
		})
	}
	return "done", nil
}

func (p *headlessUsageProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

type headlessHistoryProbeProvider struct {
	responses []string
	histories [][]api.Message
	callCount int
}

func (p *headlessHistoryProbeProvider) Name() string { return "gemini" }

func (p *headlessHistoryProbeProvider) SupportsImages() bool { return false }

func (p *headlessHistoryProbeProvider) IsFunctionCallingEnabled() bool { return true }

func (p *headlessHistoryProbeProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	p.histories = append(p.histories, cloneHeadlessHistory(history))
	if p.callCount >= len(p.responses) {
		return p.responses[len(p.responses)-1], nil
	}
	resp := p.responses[p.callCount]
	p.callCount++
	return resp, nil
}

func (p *headlessHistoryProbeProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

func cloneHeadlessHistory(history []api.Message) []api.Message {
	cloned := make([]api.Message, len(history))
	for i, msg := range history {
		cloned[i] = msg
		if len(msg.ToolCalls) > 0 {
			cloned[i].ToolCalls = append([]api.OpenAIToolCall(nil), msg.ToolCalls...)
		}
	}
	return cloned
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

func TestRunHeadlessWithConfig_CollectsTokenUsageAndCost(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := &headlessUsageProvider{}
	result := RunHeadlessWithConfig(context.Background(), "probe", "gpt-5.4-nano", provider, newProjectMapDisabledConfig())
	if result.Status != "success" {
		t.Fatalf("result.Status = %q, want success", result.Status)
	}
	if result.Tokens == nil {
		t.Fatal("result.Tokens = nil, want usage summary")
	}
	if result.Tokens.Input != 1000 {
		t.Fatalf("result.Tokens.Input = %d, want 1000", result.Tokens.Input)
	}
	if result.Tokens.Cached != 200 {
		t.Fatalf("result.Tokens.Cached = %d, want 200", result.Tokens.Cached)
	}
	if result.Tokens.Output != 300 {
		t.Fatalf("result.Tokens.Output = %d, want 300", result.Tokens.Output)
	}
	if result.Tokens.Thinking != 50 {
		t.Fatalf("result.Tokens.Thinking = %d, want 50", result.Tokens.Thinking)
	}
	if result.Tokens.Total != 1350 {
		t.Fatalf("result.Tokens.Total = %d, want 1350", result.Tokens.Total)
	}

	expectedCost := CalculateRequestCostWithCache("openai", "gpt-5.4-nano", api.Usage{
		InputTokens:       1000,
		CachedInputTokens: 200,
		OutputTokens:      300,
		ThinkingTokens:    50,
	})
	if result.Cost != expectedCost {
		t.Fatalf("result.Cost = %f, want %f", result.Cost, expectedCost)
	}
}

func TestRunHeadlessWithConfig_ProjectMapAddsQueryFocusOverlay(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep (rg) not available")
	}

	t.Setenv("HOME", t.TempDir())

	root := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	nested := filepath.Join(root, "internal", "agent", "compress.go")
	if err := os.MkdirAll(filepath.Dir(nested), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nested, []byte("package agent\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	provider := &headlessToolSetProbeProvider{}

	result := RunHeadlessWithConfig(context.Background(), "internal/agent/compress.go を見て", "gpt-5.4", provider, cfg)
	if result.Status != "success" {
		t.Fatalf("result.Status = %q, want success", result.Status)
	}
	if !strings.Contains(provider.systemPrompt, "## Project Map") {
		t.Fatalf("expected stable project map section in headless prompt:\n%s", provider.systemPrompt)
	}
	if !strings.Contains(provider.systemPrompt, "Focus files for current task:") {
		t.Fatalf("expected focus overlay in headless prompt:\n%s", provider.systemPrompt)
	}
	if !strings.Contains(provider.systemPrompt, "internal/agent/compress.go") {
		t.Fatalf("expected headless system prompt to include query focus file:\n%s", provider.systemPrompt)
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

func TestRunHeadlessWithConfig_UsesFunctionCallingHistoryForToolLoop(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	dir := testSubDir(t)
	testFile := fmt.Sprintf("%s/probe.txt", dir)
	if err := os.WriteFile(testFile, []byte("hello from headless\n"), 0644); err != nil {
		t.Fatal(err)
	}

	provider := &headlessHistoryProbeProvider{
		responses: []string{
			fmt.Sprintf(`{"tool": "gather_context", "args": {"query": %q}}`, testFile),
			"done",
		},
	}

	result := RunHeadlessWithConfig(context.Background(), "Read the probe file", "test-model", provider, newProjectMapDisabledConfig())
	if result.Status != "success" {
		t.Fatalf("RunHeadlessWithConfig() status = %q, want success", result.Status)
	}
	if len(provider.histories) != 2 {
		t.Fatalf("provider histories = %d, want 2", len(provider.histories))
	}

	secondHistory := provider.histories[1]
	if len(secondHistory) != 3 {
		t.Fatalf("second history length = %d, want 3", len(secondHistory))
	}
	if secondHistory[0].Role != "user" {
		t.Fatalf("history[0].Role = %q, want user", secondHistory[0].Role)
	}
	if secondHistory[1].Role != "assistant" {
		t.Fatalf("history[1].Role = %q, want assistant", secondHistory[1].Role)
	}
	if len(secondHistory[1].ToolCalls) != 1 {
		t.Fatalf("history[1].ToolCalls length = %d, want 1", len(secondHistory[1].ToolCalls))
	}

	toolCall := secondHistory[1].ToolCalls[0]
	if toolCall.ID == "" {
		t.Fatal("history[1].ToolCalls[0].ID is empty, want rescue tool_call_id")
	}
	if toolCall.Function.Name != "gather_context" {
		t.Errorf("history[1].ToolCalls[0].Function.Name = %q, want gather_context", toolCall.Function.Name)
	}

	if secondHistory[2].Role != "tool" {
		t.Fatalf("history[2].Role = %q, want tool", secondHistory[2].Role)
	}
	if secondHistory[2].ToolCallID != toolCall.ID {
		t.Errorf("history[2].ToolCallID = %q, want %q", secondHistory[2].ToolCallID, toolCall.ID)
	}
	if secondHistory[2].ToolName != "gather_context" {
		t.Errorf("history[2].ToolName = %q, want gather_context", secondHistory[2].ToolName)
	}
	if !strings.Contains(secondHistory[2].Content, "hello from headless") {
		t.Errorf("history[2].Content = %q, want gather_context output", secondHistory[2].Content)
	}
}

func TestRunHeadless_CallsCleanup(t *testing.T) {
	var called atomic.Int32
	cleanupHook = func() { called.Add(1) }
	defer func() { cleanupHook = nil }()

	provider := &mockProvider{name: "test"}
	_ = RunHeadlessWithConfig(context.Background(), "hello", "test-model", provider, newProjectMapDisabledConfig())

	if called.Load() != 1 {
		t.Errorf("Cleanup was called %d times, want 1", called.Load())
	}
}

func TestRunHeadless_CallsCleanupOnError(t *testing.T) {
	var called atomic.Int32
	cleanupHook = func() { called.Add(1) }
	defer func() { cleanupHook = nil }()

	provider := &mockErrorProvider{}
	result := RunHeadlessWithConfig(context.Background(), "hello", "test-model", provider, newProjectMapDisabledConfig())

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
		_ = RunHeadlessWithConfig(context.Background(), "hello", "test-model", provider, newProjectMapDisabledConfig())
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
		res := RunHeadlessWithConfig(context.Background(), "test query", "mock-model", provider, newProjectMapDisabledConfig())
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
