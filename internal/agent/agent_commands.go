package agent

import (
	"fmt"
	"io"
	"os"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
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
	if hasTUIRuntimePrompter(runtime) {
		return true
	}
	result := common.ConfirmInteractiveWithIO(runtime.PromptIO(), prompt)
	if result.Action == "comment" {
		yellow.Fprintln(runtime.Output(), "⚠️  Comment mode is for AI tool confirmations only. Treating as cancel.")
		return false
	}
	return result.Action == "yes"
}

type tuiRuntimePrompter interface {
	isTUIRuntimePrompter()
}

func hasTUIRuntimePrompter(runtime *ui.Runtime) bool {
	if runtime == nil {
		return false
	}
	_, ok := runtime.Prompter().(tuiRuntimePrompter)
	return ok
}

// handleSpecialCommand は特殊コマンドを処理
func handleSpecialCommand(input string, agent *Agent) bool {
	return handleSpecialCommandForSurface(input, agent, commandcatalog.CommandSurfaceClassic)
}

func handleSpecialCommandForSurface(input string, agent *Agent, commandSurface commandcatalog.CommandSurface) bool {
	invocation, ok := commandruntime.Parse(input)
	if !ok {
		return false
	}
	if cmdInfo, known := commandcatalog.Find(invocation.Command); known && !cmdInfo.SupportsSurface(commandSurface) {
		return handleUnsupportedCommandSurface(invocation, cmdInfo, agent, commandSurface)
	}
	handler, ok := specialCommandRegistry(agent, commandSurface)[invocation.Command]
	if !ok {
		return false
	}
	return handler(invocation.Args)
}

func handleUnsupportedCommandSurface(invocation commandruntime.Invocation, cmdInfo commandcatalog.CommandInfo, agent *Agent, commandSurface commandcatalog.CommandSurface) bool {
	if len(invocation.Args) != 0 {
		return false
	}

	yellow.Fprintf(agent.output(), "⚠️  %s is available in %s mode only.\n", invocation.Command, commandSurfaceHint(cmdInfo))
	if commandSurface == commandcatalog.CommandSurfaceClassic && cmdInfo.SupportsSurface(commandcatalog.CommandSurfaceTUI) {
		yellow.Fprintln(agent.output(), "   Run without --no-tui to use the TUI command.")
	}
	return true
}

func commandSurfaceHint(cmdInfo commandcatalog.CommandInfo) string {
	if cmdInfo.SupportsSurface(commandcatalog.CommandSurfaceTUI) {
		return "TUI"
	}
	if cmdInfo.SupportsSurface(commandcatalog.CommandSurfaceClassic) {
		return "classic"
	}
	return "another"
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
