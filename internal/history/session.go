package history

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/api"
)

const toolExecutionEntryType = "tool_execution"

// Session は会話セッションを表す
type Session struct {
	ID                        string
	Model                     string
	ProviderName              string
	ProviderConfigKey         string
	StartTime                 time.Time
	LastModified              time.Time
	Messages                  []MessageEntry
	CompactedItems            []CompactedItem `json:"compacted_items,omitempty"`   // Compact API 圧縮済みアイテム
	IsCompactedMode           bool            `json:"is_compacted_mode,omitempty"` // 圧縮モードフラグ
	ResponseID                string          `json:"response_id,omitempty"`       // OpenAI Responses API の継続コンテキスト用レスポンスID
	ResponseModel             string          `json:"response_model,omitempty"`
	ResponseProviderName      string          `json:"response_provider_name,omitempty"`
	ResponseProviderConfigKey string          `json:"response_provider_config_key,omitempty"`
	persistedCount            int
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
	Timestamp        time.Time                `json:"timestamp"`
	Role             string                   `json:"role"`
	Content          string                   `json:"content"`
	Model            string                   `json:"model,omitempty"`
	ResponseID       string                   `json:"response_id,omitempty"`       // OpenAI Responses API の ID
	ToolCalls        []api.OpenAIToolCall     `json:"tool_calls,omitempty"`        // FC: assistant のツール呼び出し
	ToolCallID       string                   `json:"tool_call_id,omitempty"`      // FC: tool レスポンスの呼び出しID
	ToolName         string                   `json:"tool_name,omitempty"`         // FC: ツール名（Gemini 用）
	ProviderMetadata *MessageProviderMetadata `json:"provider_metadata,omitempty"` // request payload には出さない provider 専用 state
	EntryType        string                   `json:"entry_type,omitempty"`        // "tool_execution" は監査用
	ToolExecution    *ToolExecutionEntry      `json:"tool_execution,omitempty"`    // ツール実行の監査情報
}

// MessageProviderMetadata は会話再開時に必要な provider 専用 state を保存する。
type MessageProviderMetadata struct {
	AnthropicContentBlocks  []api.AnthropicContentBlock  `json:"anthropic_content_blocks,omitempty"`
	AnthropicThinkingBlocks []api.AnthropicThinkingBlock `json:"anthropic_thinking_blocks,omitempty"` // legacy metadata
}

// ToolExecutionEntry はツール実行の監査情報です。
type ToolExecutionEntry struct {
	Name          string            `json:"name"`
	Args          map[string]string `json:"args,omitempty"`
	ResultPreview string            `json:"result_preview,omitempty"`
	Success       bool              `json:"success"`
}

// SessionMetadata はセッション一覧用のメタデータ
type SessionMetadata struct {
	ID                        string    `json:"session_id"`
	Model                     string    `json:"model"`
	ProviderName              string    `json:"provider_name,omitempty"`
	ProviderConfigKey         string    `json:"provider_config_key,omitempty"`
	StartTime                 time.Time `json:"start_time"`
	LastModified              time.Time `json:"last_modified"`
	MessageCount              int       `json:"message_count"`
	Preview                   string    `json:"preview"`
	ResponseID                string    `json:"response_id,omitempty"`
	ResponseContextVersion    int       `json:"response_context_version,omitempty"`
	ResponseModel             string    `json:"response_model,omitempty"`
	ResponseProviderName      string    `json:"response_provider_name,omitempty"`
	ResponseProviderConfigKey string    `json:"response_provider_config_key,omitempty"`
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
		Timestamp:        time.Now(),
		Role:             msg.Role,
		Content:          msg.Content,
		Model:            model,
		ToolCalls:        msg.ToolCalls,
		ToolCallID:       msg.ToolCallID,
		ToolName:         msg.ToolName,
		ProviderMetadata: providerMetadataFromAPIMessage(msg),
	})
	s.LastModified = time.Now()
}

// AddToolExecution はツール実行の監査情報をセッションに追加します。
func (s *Session) AddToolExecution(toolName string, args map[string]string, result string, success bool, model string) {
	s.Messages = append(s.Messages, MessageEntry{
		Timestamp: time.Now(),
		Role:      "tool",
		Model:     model,
		EntryType: toolExecutionEntryType,
		ToolExecution: &ToolExecutionEntry{
			Name:          toolName,
			Args:          cloneStringMap(args),
			ResultPreview: truncateRunes(result, 200),
			Success:       success,
		},
	})
	s.LastModified = time.Now()
}

// ToAPIMessages はAPI形式に変換
func (s *Session) ToAPIMessages() []api.Message {
	msgs := make([]api.Message, 0, len(s.Messages))
	for _, m := range s.Messages {
		if m.EntryType == toolExecutionEntryType {
			continue
		}
		msg := api.Message{
			Role:       m.Role,
			Content:    m.Content,
			ToolCalls:  m.ToolCalls,
			ToolCallID: m.ToolCallID,
			ToolName:   m.ToolName,
		}
		if m.ProviderMetadata != nil {
			if len(m.ProviderMetadata.AnthropicContentBlocks) > 0 {
				msg.SetAnthropicContentBlocks(m.ProviderMetadata.AnthropicContentBlocks)
			} else {
				msg.SetAnthropicThinkingBlocks(m.ProviderMetadata.AnthropicThinkingBlocks)
			}
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

func providerMetadataFromAPIMessage(msg api.Message) *MessageProviderMetadata {
	contentBlocks := msg.AnthropicContentBlocks()
	if len(contentBlocks) > 0 {
		return &MessageProviderMetadata{
			AnthropicContentBlocks: contentBlocks,
		}
	}

	thinkingBlocks := msg.AnthropicThinkingBlocks()
	if len(thinkingBlocks) == 0 {
		return nil
	}
	return &MessageProviderMetadata{
		AnthropicThinkingBlocks: thinkingBlocks,
	}
}

func (s *Session) unsavedMessages() []MessageEntry {
	if s == nil {
		return nil
	}
	if s.persistedCount < 0 {
		s.persistedCount = 0
	}
	if s.persistedCount >= len(s.Messages) {
		return nil
	}
	return s.Messages[s.persistedCount:]
}

func (s *Session) markPersisted() {
	if s == nil {
		return
	}
	s.persistedCount = len(s.Messages)
}

func (s *Session) TruncateMessages(count int) bool {
	if s == nil {
		return false
	}
	if count < 0 {
		count = 0
	}
	if count >= len(s.Messages) {
		return false
	}

	s.Messages = append([]MessageEntry(nil), s.Messages[:count]...)
	if s.persistedCount > len(s.Messages) {
		s.persistedCount = len(s.Messages)
	}
	s.LastModified = time.Now()
	return true
}

func (s *Session) ResetConversation() {
	if s == nil {
		return
	}

	s.Messages = nil
	s.CompactedItems = nil
	s.IsCompactedMode = false
	clearSavedResponseContext(s)
	s.persistedCount = 0
	s.LastModified = time.Now()
}

func (s *Session) conversationMessageCount() int {
	count := 0
	for _, msg := range s.Messages {
		if msg.EntryType == toolExecutionEntryType {
			continue
		}
		count++
	}
	return count
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}
