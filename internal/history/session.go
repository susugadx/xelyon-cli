package history

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/api"
)

const (
	toolExecutionEntryType  = "tool_execution"
	compactedStateEntryType = "compacted_state"
)

var sessionIDCounter uint64

// Session は会話セッションを表す
type Session struct {
	ID                        string
	Model                     string
	ProviderName              string
	ProviderConfigKey         string
	WorkingDir                string
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
	rewriteRequired           bool
}

// CompactedItem は Compact API の圧縮済み input item の保存用エイリアス。
type CompactedItem = api.InputItem

// MessageEntry はタイムスタンプ付きメッセージ
type MessageEntry struct {
	Timestamp        time.Time                `json:"timestamp"`
	Role             string                   `json:"role"`
	Content          string                   `json:"content"`
	ReasoningContent string                   `json:"reasoning_content,omitempty"` // OpenAI互換 reasoning_content
	Model            string                   `json:"model,omitempty"`
	ResponseID       string                   `json:"response_id,omitempty"`       // OpenAI Responses API の ID
	ToolCalls        []api.OpenAIToolCall     `json:"tool_calls,omitempty"`        // FC: assistant のツール呼び出し
	ToolCallID       string                   `json:"tool_call_id,omitempty"`      // FC: tool レスポンスの呼び出しID
	ToolName         string                   `json:"tool_name,omitempty"`         // FC: ツール名（Gemini 用）
	ProviderMetadata *MessageProviderMetadata `json:"provider_metadata,omitempty"` // request payload には出さない provider 専用 state
	EntryType        string                   `json:"entry_type,omitempty"`        // "tool_execution" は監査用
	ToolExecution    *ToolExecutionEntry      `json:"tool_execution,omitempty"`    // ツール実行の監査情報
	CompactedItems   []CompactedItem          `json:"compacted_items,omitempty"`   // Compact API 圧縮 state
	IsCompactedMode  bool                     `json:"is_compacted_mode,omitempty"` // Compact API 圧縮 mode
}

// MessageProviderMetadata は会話再開時に必要な provider 専用 state を保存する。
type MessageProviderMetadata struct {
	AnthropicContentBlocks  []api.AnthropicContentBlock  `json:"anthropic_content_blocks,omitempty"`
	AnthropicThinkingBlocks []api.AnthropicThinkingBlock `json:"anthropic_thinking_blocks,omitempty"` // legacy metadata
	OpenAIResponsesItems    []api.InputItem              `json:"openai_responses_items,omitempty"`
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
	WorkingDir                string    `json:"working_dir,omitempty"`
	StartTime                 time.Time `json:"start_time"`
	LastModified              time.Time `json:"last_modified"`
	MessageCount              int       `json:"message_count"`
	CompactedItemCount        int       `json:"compacted_item_count,omitempty"`
	IsCompactedMode           bool      `json:"is_compacted_mode,omitempty"`
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
		ID:           newSessionID(now),
		Model:        model,
		WorkingDir:   currentWorkingDirForSession(),
		StartTime:    now,
		LastModified: now,
		Messages:     []MessageEntry{},
	}
}

func newSessionID(now time.Time) string {
	sequence := atomic.AddUint64(&sessionIDCounter, 1)
	return fmt.Sprintf("%d-%d", now.UnixNano(), sequence)
}

func currentWorkingDirForSession() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	if abs, err := filepath.Abs(cwd); err == nil {
		cwd = abs
	}
	return filepath.Clean(cwd)
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
	s.AddMessageFromAPIWithStoredContent(msg, msg.Content, model)
}

// AddMessageFromAPIWithStoredContent は api.Message のメタデータを保持し、保存 content だけ指定値に差し替える。
func (s *Session) AddMessageFromAPIWithStoredContent(msg api.Message, content, model string) {
	now := time.Now()
	s.Messages = append(s.Messages, newMessageEntryFromAPIWithStoredContent(msg, content, model, now))
	s.LastModified = now
}

// ReplaceMessagesFromAPI は session の会話メッセージを API message 群で置き換える。
func (s *Session) ReplaceMessagesFromAPI(messages []api.Message, model string) {
	if s == nil {
		return
	}

	now := time.Now()
	s.Messages = make([]MessageEntry, 0, len(messages))
	for _, msg := range messages {
		s.Messages = append(s.Messages, newMessageEntryFromAPI(msg, model, now))
	}
	s.persistedCount = 0
	s.requireRewrite()
	s.LastModified = now
}

// SetCompactedState は Compact API の圧縮 state を session に保存する。
func (s *Session) SetCompactedState(items []CompactedItem, enabled bool) {
	if s == nil {
		return
	}

	s.CompactedItems = cloneCompactedItems(items)
	s.IsCompactedMode = enabled && len(s.CompactedItems) > 0
	s.requireRewrite()
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
	s.requireRewrite()
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
	s.requireRewrite()
	s.LastModified = time.Now()
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

func cloneCompactedItems(src []CompactedItem) []CompactedItem {
	return api.CloneInputItems(src)
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
