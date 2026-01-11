package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
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
	messages := []api.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "Can you help me?"},
	}

	prompt := buildSummaryPrompt(messages)

	// プロンプトに必要な要素が含まれているか確認
	if prompt == "" {
		t.Error("buildSummaryPrompt() returned empty string")
	}

	// ユーザーメッセージが含まれているか
	if !stringContains(prompt, "Hello") {
		t.Error("buildSummaryPrompt() should contain user message 'Hello'")
	}

	// アシスタントメッセージが含まれているか
	if !stringContains(prompt, "Hi there!") {
		t.Error("buildSummaryPrompt() should contain assistant message 'Hi there!'")
	}

	// 指示文が含まれているか
	if !stringContains(prompt, "要約") && !stringContains(prompt, "サマリー") {
		t.Error("buildSummaryPrompt() should contain instruction keywords")
	}
}

func TestBuildSummaryPrompt_LongMessage(t *testing.T) {
	longContent := ""
	for i := 0; i < 600; i++ {
		longContent += "a"
	}

	messages := []api.Message{
		{Role: "user", Content: longContent},
	}

	prompt := buildSummaryPrompt(messages)

	// 長いメッセージは500文字で省略される
	if !stringContains(prompt, "省略") {
		t.Error("buildSummaryPrompt() should truncate long messages and show '省略'")
	}
}

func TestAgent_CompressHistory_TooShort(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider)

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
	agent := NewAgent("test-model", provider)

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
