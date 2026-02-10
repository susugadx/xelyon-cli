package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/prompt"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		messages []api.Message
		want     int
	}{
		{
			name:     "empty messages",
			messages: []api.Message{},
			want:     0,
		},
		{
			name: "single short message",
			messages: []api.Message{
				{Role: "user", Content: "Hello"},
			},
			want: 1, // 5 chars / 3 = 1.6 → 1
		},
		{
			name: "multiple messages",
			messages: []api.Message{
				{Role: "user", Content: "Hello"},       // 5 chars
				{Role: "assistant", Content: "Hi!"},    // 3 chars
				{Role: "user", Content: "How are you"}, // 11 chars
			},
			want: 6, // (5+3+11) / 3 = 6.3 → 6
		},
		{
			name: "long message",
			messages: []api.Message{
				{Role: "user", Content: "This is a very long message with many characters for testing purposes"},
			},
			want: 23, // 69 chars / 3 = 23
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateTokens(tt.messages)
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
