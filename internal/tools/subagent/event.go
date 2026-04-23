package subagent

import (
	"context"
	"fmt"
	"strings"
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

// EmitCompletionEvent はサブエージェント完了イベントを送信します。
func EmitCompletionEvent(ctx context.Context, status string, result *RunResult) {
	EmitEvent(ctx, SubAgentEvent{
		Tool:    "_completed",
		Phase:   "end",
		Output:  formatCompletionEventOutput(status, result),
		Success: status == "completed",
	})
}

func formatCompletionEventOutput(status string, result *RunResult) string {
	if result == nil {
		if status == "" {
			return "unknown"
		}
		return status
	}
	if status != "completed" {
		if result.ErrorMessage != "" {
			return result.ErrorMessage
		}
		if status != "" {
			return status
		}
		return "error"
	}

	parts := make([]string, 0, 2)
	if result.ToolExecutions > 0 {
		label := "tools"
		if result.ToolExecutions == 1 {
			label = "tool"
		}
		parts = append(parts, fmt.Sprintf("%d %s", result.ToolExecutions, label))
	}
	if result.DurationMs > 0 {
		parts = append(parts, formatEventDuration(result.DurationMs))
	}
	if len(parts) == 0 {
		return "completed"
	}
	return fmt.Sprintf("completed (%s)", strings.Join(parts, ", "))
}

func formatEventDuration(durationMs int64) string {
	if durationMs < 1000 {
		return fmt.Sprintf("%dms", durationMs)
	}
	seconds := float64(durationMs) / 1000
	if durationMs%1000 == 0 {
		return fmt.Sprintf("%.0fs", seconds)
	}
	return fmt.Sprintf("%.1fs", seconds)
}
