package tui

import (
	"time"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

const ChatRoleAssistantChunk = "assistant_chunk"

// ChatMessage は会話ログの1エントリ
type ChatMessage struct {
	Role      string       // "user", "assistant", "assistant_chunk", "tool_header", "system_info"
	Content   string       // テキスト内容（ANSIカラー付き可）
	Tools     []ToolResult // Phase 2 以降: ツール結果の折りたたみ/展開表示用
	Timestamp time.Time
}

// ToolStatus は TUI tool timeline の表示状態。
type ToolStatus string

const (
	ToolStatusRunning ToolStatus = "running"
	ToolStatusOK      ToolStatus = "ok"
	ToolStatusError   ToolStatus = "error"
)

// ToolResult は1つのツール実行結果
type ToolResult struct {
	Name      string // "search_code", "apply_patch", "read_file" 等
	Summary   string // 1行サマリー
	Detail    string // 展開時に表示する全文
	Collapsed bool   // true=折りたたみ、false=展開
	Error     bool   // エラーかどうか
	ID        string
	Status    ToolStatus
	Target    string
	StartedAt time.Time
	Duration  time.Duration
}

// StatusSnapshot は TUI のステータスバー用に構造化した内部 runtime 状態。
type StatusSnapshot struct {
	Provider   string
	Model      string
	Mode       string
	Tokens     string
	Cost       string
	LegacyLine string
}

// tea.Msg として使うメッセージ型

// AppendMessageMsg は会話ログにメッセージを追加するMsg
type AppendMessageMsg struct {
	Message ChatMessage
}

// AppendToolResultMsg はツール結果を追加するMsg。Phase 2 以降で使用。
type AppendToolResultMsg struct {
	Tool ToolResult
}

// StreamTextMsg はAI応答のストリーミングテキストMsg。Phase 2 以降で使用。
type StreamTextMsg struct {
	Text string
	Done bool // ストリーミング完了
}

// UpdateStatusMsg はステータスバー更新Msg
type UpdateStatusMsg struct {
	Line string
}

// AgentDoneMsg はagent.chat()の完了通知
type AgentDoneMsg struct {
	Error     error
	ErrorKind AgentErrorKind
}

// OpenPromptMsg は TUI prompt modal を開くMsg。
type OpenPromptMsg struct {
	ID      uint64
	Request ui.PromptRequest
	Respond chan<- ui.PromptResponse
}

// CancelPromptMsg は待機中の prompt modal をキャンセルするMsg。
type CancelPromptMsg struct {
	ID uint64
}
