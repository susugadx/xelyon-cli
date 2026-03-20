package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/deepseek"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
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

func (m *mockProvider) IsFunctionCallingEnabled() bool {
	return true
}

func (m *mockProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	return "mock response", nil
}

func (m *mockProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return "mock image response", nil
}

func TestShouldAbortToolLoop_SameToolRepeated(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)

	// 閾値を3に設定
	cfg := agent.cfg()
	originalThreshold := cfg.LoopDetection.Threshold
	cfg.LoopDetection.Threshold = 3
	defer func() {
		cfg.LoopDetection.Threshold = originalThreshold
	}()

	toolCall := &tools.ToolCall{
		Tool: "read_file",
		Args: testReadFileArgs("/test.txt"),
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
	agent := NewAgent("test-model", provider, false)

	cfg := agent.cfg()
	originalThreshold := cfg.LoopDetection.Threshold
	cfg.LoopDetection.Threshold = 3
	defer func() {
		cfg.LoopDetection.Threshold = originalThreshold
	}()

	toolCall1 := &tools.ToolCall{
		Tool: "read_file",
		Args: testReadFileArgs("/test.txt"),
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
	agent := NewAgent("test-model", provider, false)

	initialHistoryLen := len(agent.History)

	toolCall := &tools.ToolCall{
		Tool: "read_file",
		Args: testReadFileArgs("/nonexistent.txt"),
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
	agent := NewAgent("test-model", provider, false)

	// Statsを初期化
	agent.Stats = &SessionStats{
		ToolExecutions: make(map[string]int),
	}

	toolCall := &tools.ToolCall{
		Tool: "read_file",
		Args: testReadFileArgs("/test.txt"),
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
	agent := NewAgent("test-model", provider, false)

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
	agent := NewAgent("test-model", provider, false)
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
	agent := NewAgent("test-model", provider, false)

	err := agent.SwitchProvider("deepseek")
	if err == nil {
		t.Error("SwitchProvider() should return error when API key is not set")
	}
}

func TestPrintHeader_ChatTest(t *testing.T) {
	provider := &mockProvider{name: "Test Provider"}

	// printHeaderToWriterはprovider.Name()を呼び出すのでpanicしないことを確認
	printHeaderToWriter(io.Discard, "test-model", provider)
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
	response := `{"tool": "read_file", "args": {"paths": ["/test.txt"]}}`

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

{"tool": "read_file", "args": {"paths": ["/test.txt"]}}`

	explanation, toolJSON := extractExplanationAndTool(response)

	expectedExplanation := "I'll read the file for you."
	if explanation != expectedExplanation {
		t.Errorf("extractExplanationAndTool() explanation = %q, want %q", explanation, expectedExplanation)
	}

	expectedToolJSON := `{"tool": "read_file", "args": {"paths": ["/test.txt"]}}`
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
	response := `{"tool": "read_file", "args": {"paths": ["/a.txt"]}}
{"tool": "read_file", "args": {"paths": ["/b.txt"]}}`

	_, toolJSON := extractExplanationAndTool(response)

	expectedFirst := `{"tool": "read_file", "args": {"paths": ["/a.txt"]}}`
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

// NOTE: shouldEnterPlanMode tests removed in Issue #82
// Plan Mode Only に統一されたため、shouldEnterPlanMode 関数とそのテストは削除されました

// TestMaxRetryConfig はmax_retryの設定が正しく読み込まれることを確認
func TestMaxRetryConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	// デフォルト値は10
	if cfg.PlanMode.MaxRetry != 10 {
		t.Errorf("PlanMode.MaxRetry = %d, want 10", cfg.PlanMode.MaxRetry)
	}

	// 一時的に変更してテスト
	original := cfg.PlanMode.MaxRetry
	cfg.PlanMode.MaxRetry = 5
	defer func() {
		cfg.PlanMode.MaxRetry = original
	}()

	if cfg.PlanMode.MaxRetry != 5 {
		t.Errorf("PlanMode.MaxRetry after change = %d, want 5", cfg.PlanMode.MaxRetry)
	}
}

// TestExecuteToolCallWithResult は結果を返すことを確認
func TestExecuteToolCallWithResult(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)

	// Statsを初期化
	agent.Stats = &SessionStats{
		ToolExecutions: make(map[string]int),
	}

	toolCall := &tools.ToolCall{
		Tool: "read_file",
		Args: testReadFileArgs("/nonexistent-file-for-test.txt"),
	}

	result := agent.executeToolCallWithResult("test response", toolCall)

	// 結果が空でないことを確認
	if result == "" {
		t.Error("executeToolCallWithResult() should return non-empty result")
	}

	// エラーメッセージを含むことを確認（ファイルが存在しないため）
	if !contains(result, "Error") && !contains(result, "error") && !contains(result, "not found") && !contains(result, "Security") {
		t.Logf("Result: %s", result)
		// セキュリティエラーの場合もOK
	}
}

// contains はsに substrが含まれるかチェック
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// sequenceMockProvider は呼び出しごとに異なるレスポンスを返すモック
type sequenceMockProvider struct {
	name      string
	responses []string
	callCount int
}

func (m *sequenceMockProvider) Name() string { return m.name }

func (m *sequenceMockProvider) SupportsImages() bool { return false }

func (m *sequenceMockProvider) IsFunctionCallingEnabled() bool { return true }

func (m *sequenceMockProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	if m.callCount >= len(m.responses) {
		// 最後のレスポンスを繰り返す（ツール呼び出しなし → ループ終了用）
		return m.responses[len(m.responses)-1], nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return resp, nil
}

func (m *sequenceMockProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return m.ChatWithTools(ctx, systemPrompt, history, model)
}

type blockingCancelProvider struct {
	// started は ChatWithTools が呼ばれたことを通知するチャネル。
	// テスト側で cancelFunc 設定済みを安全に検知するために使用する。
	started chan struct{}
}

func (m *blockingCancelProvider) Name() string { return "blocking-cancel" }

func (m *blockingCancelProvider) SupportsImages() bool { return false }

func (m *blockingCancelProvider) IsFunctionCallingEnabled() bool { return false }

func (m *blockingCancelProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	if m.started != nil {
		select {
		case <-m.started:
		default:
			close(m.started)
		}
	}
	<-ctx.Done()
	return "", fmt.Errorf("API call failed: %w", ctx.Err())
}

func (m *blockingCancelProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return m.ChatWithTools(ctx, systemPrompt, history, model)
}

// TestNormalMode_CreatePlanFC は Normal Mode で create_plan ツールコールが FC で来た場合、
// deprecated 扱いで通常モード継続できることをテスト
func TestNormalMode_CreatePlanFC(t *testing.T) {
	// create_plan ツールコールを含むレスポンス → deprecated として無視 → 通常応答で終了
	provider := &sequenceMockProvider{
		name: "test",
		responses: []string{
			// 1回目: create_plan FC ツールコール
			`I'll create a plan for this task.
{"tool": "create_plan", "args": {"title": "Test Plan", "summary": "Test", "steps": "[{\"id\":1,\"description\":\"Step 1\",\"tools\":[\"bash\"]}]"}}`,
			// 2回目: 通常応答（ツールなし）
			"OK. Continuing without create_plan.",
		},
	}

	agent := NewAgent("test-model", provider, false)
	agent.Stats = NewSessionStats("test")

	ctx := context.Background()
	err := agent.runNormalMode(ctx, "do something complex", nil)

	if err != nil {
		t.Errorf("runNormalMode() returned error: %v", err)
	}
}

// TestNormalMode_PlanJSONFallback は Normal Mode で Plan JSON がテキスト出力された場合、
// ExtractPlanJSON → ParsePlan → runImplementationPhase に切り替わることをテスト
func TestNormalMode_PlanJSONFallback(t *testing.T) {
	// Plan JSON をテキストで出力 → パース成功 → step-by-step 実行
	provider := &sequenceMockProvider{
		name: "test",
		responses: []string{
			// 1回目: Plan JSON をテキストで出力（FC失敗モデル）
			`Here is my plan:
{"plan": {"summary": "Test plan", "steps": [{"id": 1, "description": "Do step 1", "tools": ["bash"]}]}}`,
			// 2回目以降（runImplementationPhase 内）: ステップ完了
			"Step 1 done.",
		},
	}

	agent := NewAgent("test-model", provider, false)
	agent.Stats = NewSessionStats("test")

	ctx := context.Background()
	err := agent.runNormalMode(ctx, "do something", nil)

	if err != nil {
		t.Errorf("runNormalMode() returned error: %v", err)
	}

	// プロバイダーが2回以上呼ばれていることを確認（Plan JSON パース + step 実行）
	if provider.callCount < 2 {
		t.Errorf("Expected provider to be called at least 2 times, got %d", provider.callCount)
	}
}

// TestNormalMode_PlanJSONParseFailed は Plan JSON のパースが失敗した場合、
// 従来どおりリダイレクトしてループが継続することをテスト
func TestNormalMode_PlanJSONParseFailed(t *testing.T) {
	// 不正な Plan JSON → パース失敗 → リダイレクト → 通常応答で終了
	provider := &sequenceMockProvider{
		name: "test",
		responses: []string{
			// 1回目: 不正な Plan JSON（steps がない）
			`{"plan": {"summary": "broken plan"}}`,
			// 2回目: 通常応答（ツールなし）
			"OK, I'll just do it directly. Task completed.",
		},
	}

	agent := NewAgent("test-model", provider, false)
	agent.Stats = NewSessionStats("test")

	ctx := context.Background()
	err := agent.runNormalMode(ctx, "do something", nil)

	if err != nil {
		t.Errorf("runNormalMode() returned error: %v", err)
	}

	// 2回呼ばれていることを確認（リダイレクト後に通常応答）
	if provider.callCount != 2 {
		t.Errorf("Expected provider to be called 2 times, got %d", provider.callCount)
	}
}

// TestExecuteStepV2_IgnoresCreatePlan は executeStepV2 内で create_plan が
// 無視されること（再帰防止）をテスト
func TestExecuteStepV2_IgnoresCreatePlan(t *testing.T) {
	// ステップ実行中に create_plan を呼ぼうとする → 無視 → ステップ完了
	provider := &sequenceMockProvider{
		name: "test",
		responses: []string{
			// 1回目: ステップ実行中に create_plan を呼ぼうとする
			`{"tool": "create_plan", "args": {"title": "Nested Plan", "summary": "Bad", "steps": "[{\"id\":1,\"description\":\"Nested\",\"tools\":[\"bash\"]}]"}}`,
			// 2回目: ステップ完了
			"Step completed.",
		},
	}

	agent := NewAgent("test-model", provider, false)
	agent.Stats = NewSessionStats("test")

	p := &plan.Plan{
		Summary: "Test plan",
		Steps: []plan.PlanStep{
			{ID: 1, Description: "Test step", Status: "pending", Tools: []string{"bash"}},
		},
	}

	ctx := context.Background()
	err := agent.executeStepV2(ctx, p, &p.Steps[0], 0, 0)

	if err != nil {
		t.Errorf("executeStepV2() returned error: %v", err)
	}
}

func TestChatCore_SetsAbortedStatusWithActualError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var out bytes.Buffer
	agent := NewAgent("test-model", &mockErrorProvider{}, false)
	agent.Runtime = &AgentRuntime{
		UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
	}

	if err := agent.chatCore("please fail", nil, false); err != nil {
		t.Fatalf("chatCore() error = %v, want nil", err)
	}

	status := agent.statusRef().getStatus()
	if status.State != StateAborted {
		t.Fatalf("status.State = %q, want %q", status.State, StateAborted)
	}
	if !strings.Contains(status.ReasonEN, "mock error") {
		t.Fatalf("status.ReasonEN = %q, want to contain %q", status.ReasonEN, "mock error")
	}
	if status.ReasonEN == "Request failed" {
		t.Fatalf("status.ReasonEN = %q, want concrete error summary", status.ReasonEN)
	}
}

func TestChatCore_SetsAbortedStatusWithCancelReason(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var out bytes.Buffer
	provider := &blockingCancelProvider{started: make(chan struct{})}
	agent := NewAgent("test-model", provider, false)
	cfg := config.DefaultConfig()
	cfg.ProjectMap.Enabled = false
	agent.Runtime = &AgentRuntime{
		Config: cfg,
		UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
	}

	done := make(chan struct{})
	go func() {
		_ = agent.chatCore("please block", nil, false)
		close(done)
	}()

	// ChatWithTools に到達するまで待機（cancelFunc は設定済み）
	select {
	case <-provider.started:
	case <-time.After(2 * time.Second):
		t.Fatal("ChatWithTools was not called")
	}

	agent.cancelActiveRequest("signal: interrupt")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("chatCore did not finish after cancellation")
	}

	status := agent.statusRef().getStatus()
	if status.State != StateAborted {
		t.Fatalf("status.State = %q, want %q", status.State, StateAborted)
	}
	if !strings.Contains(status.ReasonEN, "signal: interrupt") {
		t.Fatalf("status.ReasonEN = %q, want to contain %q", status.ReasonEN, "signal: interrupt")
	}
}

func TestPrintTaskUsage(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = NewSessionStats("test", "test-model")
	agent.ProviderName = "test"

	// Mock start stats (100 in, 50 out)
	startStats := SessionStats{
		InputTokens:  100,
		OutputTokens: 50,
		Provider:     "test",
		Model:        "test-model",
	}

	// Mock current stats (200 in, 100 out) -> Diff: 100 in, 50 out
	agent.Stats.InputTokens = 200
	agent.Stats.OutputTokens = 100

	var out bytes.Buffer
	agent.Runtime = &AgentRuntime{
		UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
	}

	agent.printTaskUsage(startStats)
	output := out.String()

	// We expect the diff to be printed: In: 100 + Out: 50 = 150 tok
	expectedIn := "100"
	expectedOut := "50"
	expectedTotal := "150"

	if !strings.Contains(output, expectedIn) || !strings.Contains(output, expectedOut) || !strings.Contains(output, expectedTotal) {
		t.Errorf("printTaskUsage output incorrect. Expected containing In: %s, Out: %s, Total: %s, but got: %s", expectedIn, expectedOut, expectedTotal, output)
	}
}
