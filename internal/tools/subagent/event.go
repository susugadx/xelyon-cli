package subagent

import (
	"context"
	"time"
)

// SubAgentEvent はサブエージェントのリアルタイムイベントです。
type SubAgentEvent struct {
	AgentID   string    // "sub-001"
	Tool      string    // "read_file", "str_replace", etc.
	Phase     string    // "start" or "end"
	FilePath  string    // ツール対象のファイルパス
	Success   bool      // Phase=="end" 時のみ有効
	Output    string    // 結果のプレビュー（truncated）
	OldStr    string    // str_replace: 置換前テキスト
	NewStr    string    // str_replace: 置換後テキスト
	ToolIndex int       // このエージェント内でのN番目のツール (1-indexed)
	Timestamp time.Time // イベント発生時刻
}

type contextKey int

const (
	eventChKey contextKey = iota
	agentIDKey
)

// WithEventChannel は ctx に eventCh を注入します。
func WithEventChannel(ctx context.Context, ch chan<- SubAgentEvent) context.Context {
	return context.WithValue(ctx, eventChKey, ch)
}

// EventChannelFromContext は ctx から eventCh を取り出します。
func EventChannelFromContext(ctx context.Context) chan<- SubAgentEvent {
	if ch, ok := ctx.Value(eventChKey).(chan<- SubAgentEvent); ok {
		return ch
	}
	return nil
}

// WithAgentID は ctx に agentID を注入します。
func WithAgentID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, agentIDKey, id)
}

// AgentIDFromContext は ctx から agentID を取り出します。
func AgentIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(agentIDKey).(string); ok {
		return id
	}
	return ""
}

// EmitEvent はイベントを非ブロッキングで送信します。
// チャネルが nil またはフルの場合は何もしません。
func EmitEvent(ctx context.Context, event SubAgentEvent) {
	ch := EventChannelFromContext(ctx)
	if ch == nil {
		return
	}
	if event.AgentID == "" {
		event.AgentID = AgentIDFromContext(ctx)
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	select {
	case ch <- event:
	default:
		// チャネルフル時はドロップ（親UIが追いつかない場合）
	}
}
