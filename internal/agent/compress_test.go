package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/prompt"
)

// capturingMockProvider は ChatWithTools の引数をキャプチャするモック
type capturingMockProvider struct {
	capturedHistory []api.Message
}

func (m *capturingMockProvider) Name() string                   { return "test" }
func (m *capturingMockProvider) SupportsImages() bool           { return false }
func (m *capturingMockProvider) IsFunctionCallingEnabled() bool { return true }
func (m *capturingMockProvider) ChatWithTools(_ context.Context, _ string, history []api.Message, _ string) (string, error) {
	m.capturedHistory = history
	return "Summary of conversation", nil
}
func (m *capturingMockProvider) ChatWithImage(_ context.Context, _ string, _ []api.Message, _ string, _ *api.ImageData, _ string) (string, error) {
	return "", nil
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		messages []api.Message
		model    string
		want     int
	}{
		{
			name:     "empty messages",
			messages: []api.Message{},
			model:    "gpt-4o",
			want:     0,
		},
		{
			name: "single short message",
			messages: []api.Message{
				{Role: "user", Content: "Hello"},
			},
			model: "gpt-4o",
			want:  token.EstimateTokenCountForModel("gpt-4o", "Hello"),
		},
		{
			name: "multiple messages",
			messages: []api.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi!"},
				{Role: "user", Content: "How are you"},
			},
			model: "claude-sonnet-4-6",
			want: token.EstimateTokenCountForModel("claude-sonnet-4-6", "Hello") +
				token.EstimateTokenCountForModel("claude-sonnet-4-6", "Hi!") +
				token.EstimateTokenCountForModel("claude-sonnet-4-6", "How are you"),
		},
		{
			name: "long message",
			messages: []api.Message{
				{Role: "user", Content: "This is a very long message with many characters for testing purposes"},
			},
			model: "gemini-2.5-pro",
			want:  token.EstimateTokenCountForModel("gemini-2.5-pro", "This is a very long message with many characters for testing purposes"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateTokens(tt.model, tt.messages)
			if got != tt.want {
				t.Errorf("estimateTokens() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildSummaryPrompt(t *testing.T) {
	messages := []prompt.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "Can you help me?"},
	}

	result := prompt.BuildSummaryPrompt(messages, config.MessageTruncateLen)

	// プロンプトに必要な要素が含まれているか確認
	if result == "" {
		t.Error("BuildSummaryPrompt() returned empty string")
	}

	// ユーザーメッセージが含まれているか
	if !stringContains(result, "Hello") {
		t.Error("BuildSummaryPrompt() should contain user message 'Hello'")
	}

	// アシスタントメッセージが含まれているか
	if !stringContains(result, "Hi there!") {
		t.Error("BuildSummaryPrompt() should contain assistant message 'Hi there!'")
	}

	// 指示文が含まれているか
	if !stringContains(result, "Summarize") {
		t.Error("BuildSummaryPrompt() should contain 'Summarize' instruction")
	}
}

func TestBuildSummaryPrompt_LongMessage(t *testing.T) {
	longContent := ""
	for i := 0; i < 600; i++ {
		longContent += "a"
	}

	messages := []prompt.Message{
		{Role: "user", Content: longContent},
	}

	result := prompt.BuildSummaryPrompt(messages, config.MessageTruncateLen)

	// 長いメッセージは省略される（"..."で終わる）
	if !stringContains(result, "...") {
		t.Error("BuildSummaryPrompt() should truncate long messages with '...'")
	}
}

func TestAgent_CompressHistory_TooShort(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)

	// 短い履歴（3件）
	agent.History = []api.Message{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "msg2"},
		{Role: "user", Content: "msg3"},
	}

	// keepRecent=5で圧縮を試みる（履歴が短すぎてエラー）
	err := agent.CompressHistory(5)
	if err == nil {
		t.Error("CompressHistory() should return error when history is too short")
	}
}

func TestAgent_CompressHistory_Success(t *testing.T) {
	// モックプロバイダーのChatWithToolsが"Summary text"を返すようにする
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)

	// 10件の履歴を作成
	for i := 0; i < 10; i++ {
		agent.History = append(agent.History, api.Message{
			Role:    "user",
			Content: "message content",
		})
	}

	initialLen := len(agent.History)

	// 最新5件を残して圧縮（mockProviderは常に成功）
	err := agent.CompressHistory(5)
	if err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}

	// 履歴が圧縮されたことを確認（summary 1件 + 最新5件 = 6件）
	expectedLen := 6
	if len(agent.History) != expectedLen {
		t.Errorf("CompressHistory() history length = %d, want %d", len(agent.History), expectedLen)
	}

	// 最初のメッセージがsummaryメッセージであることを確認
	if agent.History[0].Role != "system" {
		t.Errorf("CompressHistory() first message role = %v, want 'system'", agent.History[0].Role)
	}

	if !stringContains(agent.History[0].Content, "Summary") && !stringContains(agent.History[0].Content, "mock") {
		t.Errorf("CompressHistory() first message should be summary, got: %s", agent.History[0].Content)
	}

	// 元の履歴より短くなっているはず
	if len(agent.History) >= initialLen {
		t.Errorf("CompressHistory() did not reduce history length: before=%d, after=%d", initialLen, len(agent.History))
	}
}

// stringContains は文字列が含まれているかをチェック
func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || stringIndexOf(s, substr) >= 0)
}

func stringIndexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestAdjustSplitForFCPairs_ToolAtKeepHead(t *testing.T) {
	// toKeep 先頭が role:"tool" → splitIdx が assistant まで巻き戻ること
	history := []api.Message{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "msg2"},
		{Role: "assistant", Content: "", ToolCalls: []api.OpenAIToolCall{
			{ID: "call_1", Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{"path":"/a.go"}`}},
		}},
		{Role: "tool", Content: "file content", ToolCallID: "call_1"},
		{Role: "user", Content: "msg3"},
	}

	// splitIdx=3 → toKeep[0] = role:"tool" → assistant(TC) まで巻き戻し → さらに assistant(TC) も toKeep に含める → splitIdx=1
	got := adjustSplitForFCPairs(history, 3)
	if got != 1 {
		t.Errorf("Expected splitIdx=1, got %d", got)
	}
}

func TestAdjustSplitForFCPairs_AssistantAtCompressTail(t *testing.T) {
	// toCompress 末尾が assistant(ToolCalls付き) → splitIdx がさらに1つ前にずれること
	history := []api.Message{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "", ToolCalls: []api.OpenAIToolCall{
			{ID: "call_1", Function: api.OpenAIToolCallFunction{Name: "bash", Arguments: `{"command":"ls"}`}},
		}},
		{Role: "tool", Content: "output", ToolCallID: "call_1"},
		{Role: "user", Content: "msg2"},
	}

	// splitIdx=2 → history[splitIdx-1] = assistant(ToolCalls) → splitIdx=1
	got := adjustSplitForFCPairs(history, 2)
	if got != 1 {
		t.Errorf("Expected splitIdx=1, got %d", got)
	}
}

func TestAdjustSplitForFCPairs_ZeroBoundary(t *testing.T) {
	// splitIdx=0 → 先頭ガードでそのまま 0 が返ること
	history := []api.Message{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "msg2"},
	}

	got := adjustSplitForFCPairs(history, 0)
	if got != 0 {
		t.Errorf("Expected splitIdx=0 (boundary guard), got %d", got)
	}
}

func TestAdjustSplitForFCPairs_MinBoundaryWithTool(t *testing.T) {
	// splitIdx=1 で tool → FC ペアごと toCompress に含めるため 2 が返ること
	history := []api.Message{
		{Role: "assistant", Content: "", ToolCalls: []api.OpenAIToolCall{
			{ID: "call_1", Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{}`}},
		}},
		{Role: "tool", Content: "result", ToolCallID: "call_1"},
		{Role: "user", Content: "msg"},
	}

	// splitIdx=1 → history[1] = role:"tool" → 巻き戻し → splitIdx=0
	// → history[0] は assistant(ToolCalls) → FC ペアごと含める → return 2
	got := adjustSplitForFCPairs(history, 1)
	if got != 2 {
		t.Errorf("Expected splitIdx=2 (FC pair preserved in toCompress), got %d", got)
	}
}

func TestAdjustSplitForFCPairs_EntireHistoryIsOneFC(t *testing.T) {
	// 全体が1つの FC ペア → 分割不可 → 0 を返す
	history := []api.Message{
		{Role: "assistant", Content: "", ToolCalls: []api.OpenAIToolCall{
			{ID: "call_1", Function: api.OpenAIToolCallFunction{Name: "bash", Arguments: `{}`}},
		}},
		{Role: "tool", Content: "result", ToolCallID: "call_1"},
	}

	got := adjustSplitForFCPairs(history, 1)
	if got != 0 {
		t.Errorf("Expected splitIdx=0 (unsplittable), got %d", got)
	}
}

func TestAdjustSplitForFCPairs_ParallelToolsAtHead(t *testing.T) {
	// history[0] = assistant(TC) with parallel tool responses
	history := []api.Message{
		{Role: "assistant", Content: "", ToolCalls: []api.OpenAIToolCall{
			{ID: "call_1", Function: api.OpenAIToolCallFunction{Name: "bash", Arguments: `{}`}},
			{ID: "call_2", Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{}`}},
		}},
		{Role: "tool", Content: "result1", ToolCallID: "call_1"},
		{Role: "tool", Content: "result2", ToolCallID: "call_2"},
		{Role: "user", Content: "next"},
	}

	// splitIdx=1 → tool → back to 0 → assistant(TC) → include FC pair → return 3
	got := adjustSplitForFCPairs(history, 1)
	if got != 3 {
		t.Errorf("Expected splitIdx=3 (parallel FC pair preserved), got %d", got)
	}
}

func TestAdjustSplitForFCPairs_NoFC(t *testing.T) {
	// FC なしの通常メッセージ → splitIdx が変わらないこと
	history := []api.Message{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "msg2"},
		{Role: "user", Content: "msg3"},
		{Role: "assistant", Content: "msg4"},
		{Role: "user", Content: "msg5"},
	}

	got := adjustSplitForFCPairs(history, 3)
	if got != 3 {
		t.Errorf("Expected splitIdx=3 (unchanged), got %d", got)
	}
}

func TestBuildFullInputItems_PrePrunesToolResults(t *testing.T) {
	// buildFullInputItems が古いツール結果を截断してから InputItem に変換することを確認
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)

	large := makeLargeContent(60)

	// keepTurns=3 より古いターンに大きい tool 結果を配置（5ターンで turn1 が古い）
	agent.History = []api.Message{
		{Role: "user", Content: "turn 1"},
		{Role: "assistant", Content: "a1"},
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "search_code"},
		{Role: "user", Content: "turn 2"},
		{Role: "assistant", Content: "a2"},
		{Role: "user", Content: "turn 3"},
		{Role: "assistant", Content: "a3"},
		{Role: "user", Content: "turn 4"},
		{Role: "assistant", Content: "a4"},
		{Role: "user", Content: "turn 5"},
		{Role: "assistant", Content: "a5"},
	}

	items := agent.buildFullInputItems()

	// items が生成されること
	if len(items) == 0 {
		t.Fatal("buildFullInputItems() returned empty")
	}

	// 古いツール結果（turn 1）が截断されていること
	// ConvertHistoryToInputItems は role="tool" を function_call_output に変換し、内容は Output フィールド
	foundTruncated := false
	for _, item := range items {
		if item.Type == "function_call_output" && strings.Contains(item.Output, "truncated") {
			foundTruncated = true
			break
		}
	}
	if !foundTruncated {
		t.Error("buildFullInputItems() should pre-prune old tool results (expected truncated marker)")
	}

	// 元の History は変更されていないこと
	if strings.Contains(agent.History[2].Content, "truncated") {
		t.Error("original History should not be modified by buildFullInputItems()")
	}
}

func TestBuildFullInputItems_CompactedModeBypassesPrune(t *testing.T) {
	// 圧縮モードの場合は compactedItems を先頭に置き、圧縮後に積まれた履歴も保持する
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)

	agent.isCompactedMode = true
	agent.compactedItems = []api.InputItem{
		{Type: "compacted", Data: "compressed-data"},
	}
	agent.History = []api.Message{{Role: "user", Content: "new turn"}}

	items := agent.buildFullInputItems()

	if len(items) != 2 {
		t.Fatalf("expected compacted item plus current history, got %d", len(items))
	}
	if items[0].Type != "compacted" {
		t.Errorf("expected type=compacted, got %q", items[0].Type)
	}
	if items[1].Role != "user" || items[1].Content != "new turn" {
		t.Fatalf("items[1] = %#v, want current history item", items[1])
	}
}

func TestCompressHistory_PrePrunesBeforeSummary(t *testing.T) {
	// CompressHistory が BuildSummaryPrompt に渡す前に古いツール結果を截断することを確認
	provider := &capturingMockProvider{}
	agent := NewAgent("test-model", provider, false)

	large := makeLargeContent(60)

	// toCompress に4ターン以上のuserメッセージを含めて、keepTurns=3 で古い tool が截断されるようにする
	// keepRecent=4 → splitIdx=14-4=10 → toCompress=history[:10], toKeep=history[10:]
	agent.History = []api.Message{
		{Role: "user", Content: "turn 1"},                                         // 0
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "search_code"}, // 1: old → truncated
		{Role: "user", Content: "turn 2"},                                         // 2
		{Role: "assistant", Content: "a2"},                                        // 3
		{Role: "user", Content: "turn 3"},                                         // 4
		{Role: "assistant", Content: "a3"},                                        // 5
		{Role: "user", Content: "turn 4"},                                         // 6
		{Role: "assistant", Content: "a4"},                                        // 7
		{Role: "user", Content: "turn 5"},                                         // 8
		{Role: "assistant", Content: "a5"},                                        // 9
		{Role: "user", Content: "turn 6"},                                         // 10 (toKeep)
		{Role: "assistant", Content: "a6"},                                        // 11
		{Role: "user", Content: "turn 7"},                                         // 12
		{Role: "assistant", Content: "a7"},                                        // 13
	}

	err := agent.CompressHistory(4)
	if err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}

	// ChatWithTools に渡されたプロンプト（history[0].Content）に "truncated" が含まれること
	if len(provider.capturedHistory) == 0 {
		t.Fatal("ChatWithTools was not called")
	}
	capturedPrompt := provider.capturedHistory[0].Content
	if !strings.Contains(capturedPrompt, "truncated") {
		t.Error("CompressHistory() should pre-prune old tool results before BuildSummaryPrompt")
	}
}

func TestCompressHistory_UsesCompressionModelDefault(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.Model = ""

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = []api.Message{
		{Role: "user", Content: "message 1"},
		{Role: "assistant", Content: "message 2"},
	}

	if err := agent.CompressHistory(1); err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}
	if provider.capturedChatModel != "gpt-5.4-mini" {
		t.Fatalf("CompressHistory() model = %q, want %q", provider.capturedChatModel, "gpt-5.4-mini")
	}
}

func TestCompressWithCompactAPI_UsesCompressionModel(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", supportsCompact: true}
	cfg := config.DefaultConfig()
	cfg.Compression.Model = ""

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = []api.Message{{Role: "user", Content: "hello"}}

	if err := agent.CompressWithCompactAPI(context.Background()); err != nil {
		t.Fatalf("CompressWithCompactAPI() error = %v", err)
	}
	if provider.capturedCompactModel != "gpt-5.4-mini" {
		t.Fatalf("CompressWithCompactAPI() model = %q, want %q", provider.capturedCompactModel, "gpt-5.4-mini")
	}
}

func TestRequestContext_IncludesCompactedInputItems(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", supportsCompact: true}
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	agent.isCompactedMode = true
	agent.compactedItems = []api.InputItem{{Type: "compacted", Data: "compact-data"}}

	got := api.CompactedInputItemsFromContext(agent.requestContext(context.Background()))
	if len(got) != 1 {
		t.Fatalf("len(CompactedInputItemsFromContext()) = %d, want 1", len(got))
	}
	if got[0].Type != "compacted" || got[0].Data != "compact-data" {
		t.Fatalf("compacted input item = %#v, want compact-data", got[0])
	}
}

func TestConvertToHistoryCompactedItems(t *testing.T) {
	tests := []struct {
		name  string
		items []api.InputItem
	}{
		{
			name:  "empty items",
			items: []api.InputItem{},
		},
		{
			name: "single item",
			items: []api.InputItem{
				{Type: "message", Role: "user", Content: "Hello"},
			},
		},
		{
			name: "multiple items with all fields",
			items: []api.InputItem{
				{Type: "message", Role: "user", Content: "Hello", ID: "1", Status: "active", Data: ""},
				{Type: "message", Role: "assistant", Content: "Hi there", ID: "2", Status: "complete", Data: ""},
				{Type: "tool_result", Role: "", Content: "result data", ID: "3", Status: "", Data: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToHistoryCompactedItems(tt.items)

			if len(result) != len(tt.items) {
				t.Errorf("convertToHistoryCompactedItems() length = %d, want %d", len(result), len(tt.items))
				return
			}

			for i, item := range tt.items {
				if result[i].Type != item.Type {
					t.Errorf("item[%d].Type = %q, want %q", i, result[i].Type, item.Type)
				}
				if result[i].Role != item.Role {
					t.Errorf("item[%d].Role = %q, want %q", i, result[i].Role, item.Role)
				}
				if result[i].Content != item.Content {
					t.Errorf("item[%d].Content = %q, want %q", i, result[i].Content, item.Content)
				}
				if result[i].ID != item.ID {
					t.Errorf("item[%d].ID = %q, want %q", i, result[i].ID, item.ID)
				}
				if result[i].Status != item.Status {
					t.Errorf("item[%d].Status = %q, want %q", i, result[i].Status, item.Status)
				}
			}
		})
	}
}

func TestConvertFromHistoryCompactedItems(t *testing.T) {
	tests := []struct {
		name  string
		items []history.CompactedItem
	}{
		{
			name:  "empty items",
			items: []history.CompactedItem{},
		},
		{
			name: "single item",
			items: []history.CompactedItem{
				{Type: "message", Role: "user", Content: "Hello"},
			},
		},
		{
			name: "multiple items with all fields",
			items: []history.CompactedItem{
				{Type: "message", Role: "user", Content: "Hello", ID: "1", Status: "active", Data: ""},
				{Type: "message", Role: "assistant", Content: "Hi there", ID: "2", Status: "complete", Data: ""},
				{Type: "tool_result", Role: "", Content: "result data", ID: "3", Status: "", Data: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertFromHistoryCompactedItems(tt.items)

			if len(result) != len(tt.items) {
				t.Errorf("convertFromHistoryCompactedItems() length = %d, want %d", len(result), len(tt.items))
				return
			}

			for i, item := range tt.items {
				if result[i].Type != item.Type {
					t.Errorf("item[%d].Type = %q, want %q", i, result[i].Type, item.Type)
				}
				if result[i].Role != item.Role {
					t.Errorf("item[%d].Role = %q, want %q", i, result[i].Role, item.Role)
				}
				if result[i].Content != item.Content {
					t.Errorf("item[%d].Content = %q, want %q", i, result[i].Content, item.Content)
				}
				if result[i].ID != item.ID {
					t.Errorf("item[%d].ID = %q, want %q", i, result[i].ID, item.ID)
				}
				if result[i].Status != item.Status {
					t.Errorf("item[%d].Status = %q, want %q", i, result[i].Status, item.Status)
				}
			}
		})
	}
}

func TestConvertRoundTrip(t *testing.T) {
	original := []api.InputItem{
		{Type: "message", Role: "user", Content: "Hello world", ID: "msg1", Status: "active"},
		{Type: "message", Role: "assistant", Content: "Hi!", ID: "msg2", Status: "complete"},
	}

	historyItems := convertToHistoryCompactedItems(original)
	result := convertFromHistoryCompactedItems(historyItems)

	if len(result) != len(original) {
		t.Fatalf("Round trip failed: length mismatch %d != %d", len(result), len(original))
	}

	for i := range original {
		if result[i].Type != original[i].Type {
			t.Errorf("Round trip item[%d].Type = %q, want %q", i, result[i].Type, original[i].Type)
		}
		if result[i].Role != original[i].Role {
			t.Errorf("Round trip item[%d].Role = %q, want %q", i, result[i].Role, original[i].Role)
		}
		if result[i].Content != original[i].Content {
			t.Errorf("Round trip item[%d].Content = %q, want %q", i, result[i].Content, original[i].Content)
		}
		if result[i].ID != original[i].ID {
			t.Errorf("Round trip item[%d].ID = %q, want %q", i, result[i].ID, original[i].ID)
		}
		if result[i].Status != original[i].Status {
			t.Errorf("Round trip item[%d].Status = %q, want %q", i, result[i].Status, original[i].Status)
		}
	}
}
