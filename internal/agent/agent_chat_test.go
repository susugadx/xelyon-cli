package agent

import (
	"context"
	"os"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// mockProvider は api.Provider のモック実装
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) SupportsImages() bool {
	return false
}

func (m *mockProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	return "mock response", nil
}

func (m *mockProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return "mock image response", nil
}

func TestShouldAbortToolLoop_SameToolRepeated(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider)

	// 閾値を3に設定
	cfg := config.GetGlobalConfig()
	originalThreshold := cfg.LoopDetection.Threshold
	cfg.LoopDetection.Threshold = 3
	defer func() {
		cfg.LoopDetection.Threshold = originalThreshold
	}()

	toolCall := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "/test.txt"},
	}

	count := 0

	// 1回目
	abort := agent.shouldAbortToolLoop(toolCall, nil, &count)
	if abort {
		t.Error("shouldAbortToolLoop() should not abort on first call")
	}
	if count != 1 {
		t.Errorf("shouldAbortToolLoop() count = %d, want 1", count)
	}

	// 2回目（同じツール）
	abort = agent.shouldAbortToolLoop(toolCall, toolCall, &count)
	if abort {
		t.Error("shouldAbortToolLoop() should not abort on second call")
	}
	if count != 2 {
		t.Errorf("shouldAbortToolLoop() count = %d, want 2", count)
	}

	// 3回目（同じツール、閾値到達）
	abort = agent.shouldAbortToolLoop(toolCall, toolCall, &count)
	if !abort {
		t.Error("shouldAbortToolLoop() should abort on third call (threshold reached)")
	}
	if count != 3 {
		t.Errorf("shouldAbortToolLoop() count = %d, want 3", count)
	}

	// Historyに警告メッセージが追加されているか確認
	if len(agent.History) == 0 {
		t.Error("shouldAbortToolLoop() should add warning message to History")
	}
}

func TestShouldAbortToolLoop_DifferentTools(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider)

	cfg := config.GetGlobalConfig()
	originalThreshold := cfg.LoopDetection.Threshold
	cfg.LoopDetection.Threshold = 3
	defer func() {
		cfg.LoopDetection.Threshold = originalThreshold
	}()

	toolCall1 := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "/test.txt"},
	}

	toolCall2 := &tools.ToolCall{
		Tool: "write_file",
		Args: map[string]string{"path": "/test.txt"},
	}

	count := 0

	// 1回目
	abort := agent.shouldAbortToolLoop(toolCall1, nil, &count)
	if abort {
		t.Error("shouldAbortToolLoop() should not abort")
	}

	// 2回目（異なるツール）
	abort = agent.shouldAbortToolLoop(toolCall2, toolCall1, &count)
	if abort {
		t.Error("shouldAbortToolLoop() should not abort when tool changes")
	}

	// カウントがリセットされるべき
	if count != 1 {
		t.Errorf("shouldAbortToolLoop() count = %d, want 1 (should reset)", count)
	}
}

func TestExecuteToolCall_UpdatesHistory(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider)

	initialHistoryLen := len(agent.History)

	toolCall := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "/nonexistent.txt"},
	}

	agent.executeToolCall("test response", toolCall)

	// Historyにassistantメッセージが追加されているべき
	if len(agent.History) != initialHistoryLen+2 {
		t.Errorf("executeToolCall() added %d messages, want 2 (assistant + tool result)", len(agent.History)-initialHistoryLen)
	}

	// 最後から2番目がassistantメッセージ
	if agent.History[len(agent.History)-2].Role != "assistant" {
		t.Errorf("executeToolCall() added message with role = %v, want 'assistant'", agent.History[len(agent.History)-2].Role)
	}
}

func TestExecuteToolCall_UpdatesStats(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider)

	// Statsを初期化
	agent.Stats = &SessionStats{
		ToolExecutions: make(map[string]int),
	}

	toolCall := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "/test.txt"},
	}

	agent.executeToolCall("test response", toolCall)

	// AssistantMessagesがインクリメントされているべき
	if agent.Stats.AssistantMessages != 1 {
		t.Errorf("executeToolCall() AssistantMessages = %d, want 1", agent.Stats.AssistantMessages)
	}

	// ToolExecutionsに記録されているべき
	if agent.Stats.ToolExecutions["read_file"] != 1 {
		t.Errorf("executeToolCall() ToolExecutions['read_file'] = %d, want 1", agent.Stats.ToolExecutions["read_file"])
	}
}

func TestAgent_Cleanup_NoStorage(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider)

	// storageとsessionをnil化
	agent.storage = nil
	agent.session = nil

	// panicしないことを確認
	agent.Cleanup()
}

func TestAgent_SwitchProvider_Success(t *testing.T) {
	// 環境変数を設定
	os.Setenv("DEEPSEEK_API_KEY", "test-key")
	defer os.Unsetenv("DEEPSEEK_API_KEY")

	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider)
	agent.ProviderName = "test"

	// Statsを初期化
	agent.Stats = NewSessionStats("test")

	err := agent.SwitchProvider("deepseek")
	if err != nil {
		t.Fatalf("SwitchProvider() error = %v", err)
	}

	if agent.ProviderName != "deepseek" {
		t.Errorf("SwitchProvider() ProviderName = %v, want 'deepseek'", agent.ProviderName)
	}

	if agent.Stats.Provider != "deepseek" {
		t.Errorf("SwitchProvider() Stats.Provider = %v, want 'deepseek'", agent.Stats.Provider)
	}

	if agent.CurrentProvider == nil {
		t.Error("SwitchProvider() CurrentProvider should not be nil")
	}
}

func TestAgent_SwitchProvider_NoAPIKey_ChatTest(t *testing.T) {
	// APIキーなし
	os.Unsetenv("DEEPSEEK_API_KEY")

	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider)

	err := agent.SwitchProvider("deepseek")
	if err == nil {
		t.Error("SwitchProvider() should return error when API key is not set")
	}
}

func TestPrintHeader_ChatTest(t *testing.T) {
	provider := &mockProvider{name: "Test Provider"}

	// printHeaderはprovider.Name()を呼び出すのでpanicしないことを確認
	printHeader("test-model", provider)
}

// extractExplanationAndTool tests

func TestExtractExplanationAndTool_NoToolCall(t *testing.T) {
	response := "This is just a plain text response with no tool calls."

	explanation, toolJSON := extractExplanationAndTool(response)

	if explanation != response {
		t.Errorf("extractExplanationAndTool() explanation = %q, want original response", explanation)
	}

	if toolJSON != "" {
		t.Errorf("extractExplanationAndTool() toolJSON = %q, want empty string", toolJSON)
	}
}

func TestExtractExplanationAndTool_OnlyToolCall(t *testing.T) {
	response := `{"tool": "read_file", "args": {"path": "/test.txt"}}`

	explanation, toolJSON := extractExplanationAndTool(response)

	if explanation != "" {
		t.Errorf("extractExplanationAndTool() explanation = %q, want empty string", explanation)
	}

	if toolJSON != response {
		t.Errorf("extractExplanationAndTool() toolJSON = %q, want %q", toolJSON, response)
	}
}

func TestExtractExplanationAndTool_BothParts(t *testing.T) {
	response := `I'll read the file for you.

{"tool": "read_file", "args": {"path": "/test.txt"}}`

	explanation, toolJSON := extractExplanationAndTool(response)

	expectedExplanation := "I'll read the file for you."
	if explanation != expectedExplanation {
		t.Errorf("extractExplanationAndTool() explanation = %q, want %q", explanation, expectedExplanation)
	}

	expectedToolJSON := `{"tool": "read_file", "args": {"path": "/test.txt"}}`
	if toolJSON != expectedToolJSON {
		t.Errorf("extractExplanationAndTool() toolJSON = %q, want %q", toolJSON, expectedToolJSON)
	}
}

func TestExtractExplanationAndTool_NestedJSON(t *testing.T) {
	response := `Here's the file operation:

{"tool": "write_file", "args": {"path": "/config.json", "content": "{\"key\": \"value\"}"}}`

	explanation, toolJSON := extractExplanationAndTool(response)

	if explanation != "Here's the file operation:" {
		t.Errorf("extractExplanationAndTool() explanation = %q, want 'Here's the file operation:'", explanation)
	}

	// The toolJSON should correctly handle nested JSON in the content
	expectedToolJSON := `{"tool": "write_file", "args": {"path": "/config.json", "content": "{\"key\": \"value\"}"}}`
	if toolJSON != expectedToolJSON {
		t.Errorf("extractExplanationAndTool() toolJSON = %q, want %q", toolJSON, expectedToolJSON)
	}
}

func TestExtractExplanationAndTool_SpacedToolPattern(t *testing.T) {
	// Test with space after opening brace
	response := `Explanation text

{ "tool": "bash", "args": {"command": "ls -la"}}`

	explanation, toolJSON := extractExplanationAndTool(response)

	if explanation != "Explanation text" {
		t.Errorf("extractExplanationAndTool() explanation = %q, want 'Explanation text'", explanation)
	}

	if toolJSON == "" {
		t.Error("extractExplanationAndTool() should detect spaced tool pattern")
	}
}

func TestExtractExplanationAndTool_EscapedQuotes(t *testing.T) {
	response := `{"tool": "str_replace", "args": {"path": "/test.txt", "old_str": "say \"hello\"", "new_str": "say \"goodbye\""}}`

	explanation, toolJSON := extractExplanationAndTool(response)

	if explanation != "" {
		t.Errorf("extractExplanationAndTool() explanation = %q, want empty string", explanation)
	}

	// Should correctly handle escaped quotes
	if toolJSON != response {
		t.Errorf("extractExplanationAndTool() toolJSON = %q, want %q", toolJSON, response)
	}
}

func TestExtractExplanationAndTool_MultipleJSONObjects(t *testing.T) {
	// Only the first tool call should be extracted
	response := `{"tool": "read_file", "args": {"path": "/a.txt"}}
{"tool": "read_file", "args": {"path": "/b.txt"}}`

	_, toolJSON := extractExplanationAndTool(response)

	expectedFirst := `{"tool": "read_file", "args": {"path": "/a.txt"}}`
	if toolJSON != expectedFirst {
		t.Errorf("extractExplanationAndTool() should extract first tool call only, got %q", toolJSON)
	}
}

func TestExtractExplanationAndTool_UnclosedBrace(t *testing.T) {
	// Test handling of malformed JSON (unclosed brace)
	response := `Explanation

{"tool": "bash", "args": {"command": "echo 'test'"`

	explanation, toolJSON := extractExplanationAndTool(response)

	if explanation != "Explanation" {
		t.Errorf("extractExplanationAndTool() explanation = %q, want 'Explanation'", explanation)
	}

	// Should return everything from tool start when brace is unclosed
	if toolJSON == "" {
		t.Error("extractExplanationAndTool() should return partial JSON when unclosed")
	}
}
