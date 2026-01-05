package history

import (
	"fmt"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// Session は会話セッションを表す
type Session struct {
	ID           string
	Model        string
	StartTime    time.Time
	LastModified time.Time
	Messages     []MessageEntry
}

// MessageEntry はタイムスタンプ付きメッセージ
type MessageEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Model     string    `json:"model,omitempty"`
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

// ToAPIMessages はAPI形式に変換
func (s *Session) ToAPIMessages() []api.Message {
	msgs := make([]api.Message, len(s.Messages))
	for i, m := range s.Messages {
		msgs[i] = api.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}
	return msgs
}
