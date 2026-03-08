package agent

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/version"
)

// printCommandHeader はコマンド出力の共通ヘッダーを表示
func printCommandHeader(title string) {
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("📊 %s\n", title)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// promptConfirm はユーザーに確認を求める（空入力は無視してリトライ）
// AI実行中のEnter押下による誤操作を防ぐ
// テストモード時は自動承認
// NOTE: ユーザーコマンド用の確認のため、コメント入力は「キャンセル」として扱う
func promptConfirm(prompt string) bool {
	// tools.ConfirmInteractive を使用してテストモード対応
	result := tools.ConfirmInteractive(prompt)
	if result.Action == "comment" {
		// ユーザーコマンドではコメントは使用しない旨を表示
		yellow.Println("⚠️  Comment mode is for AI tool confirmations only. Treating as cancel.")
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
		green.Println("🗑️  History cleared")
		return true
	case "/history":
		handleHistoryCommand(agent)
		return true
	case "/help":
		printHelp()
		return true

	case "/model":
		return handleModelCommand(agent, args)
	case "/version":
		cyan.Printf("🚀 XELYON CLI v%s\n", version.GetVersion())
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
	yellow.Println("👋 See you!")
	os.Exit(0)
}

// handleHistoryCommand は会話履歴を表示
func handleHistoryCommand(agent *Agent) {
	fmt.Printf("📜 %d messages in history\n", len(agent.History))
	for i, msg := range agent.History {
		role := "👤"
		if msg.Role == "assistant" {
			role = "🤖"
		}
		preview := msg.Content
		if len(preview) > config.HistoryPreviewLen {
			preview = preview[:config.HistoryPreviewLen] + "..."
		}
		fmt.Printf("  %d. %s %s\n", i+1, role, preview)
	}
}

// printHelp はヘルプを表示
func printHelp() {
	fmt.Print(GeneratedHelpText)
}
