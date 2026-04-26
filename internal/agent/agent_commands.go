package agent

import (
	"fmt"
	"io"
	"os"

	"github.com/susugadx/xelyon-cli/internal/commandruntime"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func printCommandHeaderToWriter(out io.Writer, title string) {
	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Fprintf(out, "📊 %s\n", title)
	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// promptConfirmWithRuntime は slash command 用の確認を runtime の入出力で行う。
// NOTE: コメント入力は AI ツール確認専用のため、ここではキャンセルとして扱う。
func promptConfirmWithRuntime(runtime *ui.Runtime, prompt string) bool {
	result := common.ConfirmInteractiveWithIO(runtime.PromptIO(), prompt)
	if result.Action == "comment" {
		yellow.Fprintln(runtime.Output(), "⚠️  Comment mode is for AI tool confirmations only. Treating as cancel.")
		return false
	}
	return result.Action == "yes"
}

// handleSpecialCommand は特殊コマンドを処理
func handleSpecialCommand(input string, agent *Agent) bool {
	return commandruntime.Dispatch(input, commandAliasesFromConfig(agent.cfg()), specialCommandRegistry(agent))
}

// splitCommand はコマンド文字列を分割
func splitCommand(input string) []string {
	return commandruntime.Split(input)
}

// handleExitCommand は終了処理を行う
func handleExitCommand(agent *Agent) {
	yellow.Fprintln(agent.output(), "👋 See you!")
	os.Exit(0)
}

// handleHistoryCommand は会話履歴を表示
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
