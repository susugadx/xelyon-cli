package agent

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// handleExitCommand は終了処理を行う。
func handleExitCommand(agent *Agent) {
	yellow.Fprintln(agent.output(), "👋 See you!")
	os.Exit(0)
}

// handleHistoryCommand は会話履歴を表示する。
func handleHistoryCommand(agent *Agent) {
	out := agent.output()
	_, _ = fmt.Fprintf(out, "📜 %d messages in history\n", len(agent.History))
	for i, msg := range agent.History {
		role := "👤"
		if msg.Role == "assistant" {
			role = "🤖"
		}
		preview := msg.Content
		if len(preview) > config.HistoryPreviewLen {
			preview = preview[:config.HistoryPreviewLen] + "..."
		}
		_, _ = fmt.Fprintf(out, "  %d. %s %s\n", i+1, role, preview)
	}
}
