package history

import (
	"fmt"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// Session は会話セッションを表す
type Session struct {
	ID              string
	Model           string
	StartTime       time.Time
	LastModified    time.Time
	Messages        []MessageEntry
	CompactedItems  []CompactedItem `json:"compacted_items,omitempty"`   // Compact API 圧縮済みアイテム
	IsCompactedMode bool            `json:"is_compacted_mode,omitempty"` // 圧縮モードフラグ
	ResponseID      string          `json:"response_id,omitempty"`       // OpenAI Responses API の最新レスポンスID
}

// CompactedItem は Compact API の圧縮済みアイテム（セッション保存用）
// api.InputItem と同一構造
type CompactedItem struct {
	Type    string      `json:"type"`              // "message" or "compacted"
	Role    string      `json:"role,omitempty"`    // "user", "assistant"
	Content interface{} `json:"content,omitempty"` // string or structured content
	ID      string      `json:"id,omitempty"`      // アシスタント応答のID
	Status  string      `json:"status,omitempty"`  // "completed"
	Data    string      `json:"data,omitempty"`    // 暗号化データ（type="compacted"の場合）
}

// MessageEntry はタイムスタンプ付きメッセージ
type MessageEntry struct {
	Timestamp  time.Time            `json:"timestamp"`
	Role       string               `json:"role"`
	Content    string               `json:"content"`
	Model      string               `json:"model,omitempty"`
	ResponseID string               `json:"response_id,omitempty"`  // OpenAI Responses API の ID
	ToolCalls  []api.OpenAIToolCall `json:"tool_calls,omitempty"`   // FC: assistant のツール呼び出し
	ToolCallID string               `json:"tool_call_id,omitempty"` // FC: tool レスポンスの呼び出しID
	ToolName   string               `json:"tool_name,omitempty"`    // FC: ツール名（Gemini 用）
}

// SessionMetadata はセッション一覧用のメタデータ
type SessionMetadata struct {
	ID           string    `json:"session_id"`
	Model        string    `json:"model"`
	StartTime    time.Time `json:"start_time"`
	LastModified time.Time `json:"last_modified"`
	MessageCount int       `json:"message_count"`
	Preview      string    `json:"preview"`
}

// NewSession は新しいセッションを作成
func NewSession(model string) *Session {
	now := time.Now()
	return &Session{
		ID:           fmt.Sprintf("%d", now.Unix()),
		Model:        model,
		StartTime:    now,
		LastModified: now,
		Messages:     []MessageEntry{},
	}
}

// AddMessage はメッセージをセッションに追加
func (s *Session) AddMessage(role, content, model string) {
	s.Messages = append(s.Messages, MessageEntry{
		Timestamp: time.Now(),
		Role:      role,
		Content:   content,
		Model:     model,
	})
	s.LastModified = time.Now()
}

// AddMessageFromAPI は api.Message から FC メタデータ付きでセッションに保存
func (s *Session) AddMessageFromAPI(msg api.Message, model string) {
	s.Messages = append(s.Messages, MessageEntry{
		Timestamp:  time.Now(),
		Role:       msg.Role,
		Content:    msg.Content,
		Model:      model,
		ToolCalls:  msg.ToolCalls,
		ToolCallID: msg.ToolCallID,
		ToolName:   msg.ToolName,
	})
	s.LastModified = time.Now()
}

// ToAPIMessages はAPI形式に変換
func (s *Session) ToAPIMessages() []api.Message {
	msgs := make([]api.Message, len(s.Messages))
	for i, m := range s.Messages {
		msgs[i] = api.Message{
			Role:       m.Role,
			Content:    m.Content,
			ToolCalls:  m.ToolCalls,
			ToolCallID: m.ToolCallID,
			ToolName:   m.ToolName,
		}
	}
	return msgs
}
