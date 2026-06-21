package mcptool

import (
	"fmt"
	"time"
)

func (w *Wrapper) callTimeoutDuration() time.Duration {
	if w.callTimeout > 0 {
		return w.callTimeout
	}
	return defaultMCPToolCallTimeout
}

func formatTimeoutDuration(d time.Duration) string {
	if d > 0 && d%time.Second == 0 {
		return fmt.Sprintf("%d seconds", int(d/time.Second))
	}
	return d.String()
}

// formatResult は結果をフォーマットする
// NOTE: 出力の切り詰めはtoken_guard.goで一元管理するため、ここでは行わない
func (w *Wrapper) formatResult(result string) string {
	// 結果が空の場合
	if result == "" {
		return "Tool executed successfully (no output)"
	}

	return result
}

// FormatResult は MCP tool のテキスト結果を表示用に整える。
func (w *Wrapper) FormatResult(result string) string {
	return w.formatResult(result)
}
