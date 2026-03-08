package agent

import (
	"fmt"
	"io"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
	"github.com/susugadx/xelyon-cli/internal/version"
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
	parts := splitCommand(input)
	if len(parts) == 0 {
		return false
	}

	cmd := resolveCommandAliasWithConfig(parts[0], agent.cfg())
	args := parts[1:]

	switch cmd {
	case "/save":
		return handleSaveCommand(agent)
	case "/load":
		return handleLoadCommand(agent, args)
	case "/sessions":
		return handleSessionsCommand(agent)
	case "/config":
		return handleConfigCommand(agent, args)
	case "/stats":
		return handleStatsCommand(agent)
	case "/status":
		return handleStatusCommand(agent)
	case "/copy":
		return handleCopyCommand(agent, args)
	case "/compress":
		return handleCompressCommand(agent, args)
	case "/use":
		return handleUseCommand(agent, args)
	case "/providers":
		return handleProvidersCommand(agent)
	case "/exit", "/quit", "/q":
		handleExitCommand(agent)
	case "/clear":
		agent.History = []api.Message{}
		green.Fprintln(agent.output(), "🗑️  History cleared")
		return true
	case "/history":
		handleHistoryCommand(agent)
		return true
	case "/help":
		printHelpToWriter(agent.output())
		return true

	case "/model":
		return handleModelCommand(agent, args)
	case "/version":
		cyan.Fprintf(agent.output(), "🚀 XELYON CLI v%s\n", version.GetVersion())
		return true
	case "/plan":
		return handlePlanCommand(agent, args)
	case "/init":
		return handleInitCommand(agent)
	case "/project":
		return handleProjectCommand(agent)
	case "/paste":
		return handlePasteCommand(agent, args)
	case "/lsp":
		return handleLSPCommand(agent, args)
	case "/tokens":
		return handleTokensCommand(agent)
	case "/think":
		return handleThinkCommand(agent, args)
	}
	return false
}

// splitCommand はコマンド文字列を分割
func splitCommand(input string) []string {
	var parts []string
	var current string
	inQuote := false
	quoteChar := rune(0)

	for _, r := range input {
		switch {
		case (r == '"' || r == '\'') && !inQuote:
			inQuote = true
			quoteChar = r
		case r == quoteChar && inQuote:
			inQuote = false
			quoteChar = 0
		case r == ' ' && !inQuote:
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		default:
			current += string(r)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
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

func printHelpToWriter(out io.Writer) {
	_, _ = fmt.Fprint(out, GeneratedHelpText)
}
