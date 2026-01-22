package agent

import (
	"context"
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

	cmd := resolveCommandAlias(parts[0])
	args := parts[1:]

	switch cmd {
	case "/save":
		return handleSaveCommand(agent)
	case "/load":
		return handleLoadCommand(agent, args)
	case "/sessions":
		return handleSessionsCommand(agent)
	case "/undo":
		return handleUndoCommand(agent, args)
	case "/changes":
		return handleChangesCommand(agent)
	case "/config":
		return handleConfigCommand(args)
	case "/stats":
		return handleStatsCommand(agent)
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
	case "/sync":
		return handleSyncCommand(agent)
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
	// Phase 3: Conversation learning (提案ベース)
	// セッション終了時に会話から「今後も守るべきルール」を抽出して XELYON.md に追記提案する
	if agent != nil {
		extractor := &LLMExtractor{Provider: agent.CurrentProvider, SystemPrompt: agent.SystemPrompt}
		ctx := context.Background()
		if _, _, err := agent.ProposeAndApplyLearning(ctx, extractor, "XELYON.md"); err != nil {
			// 失敗しても終了は継続
			yellow.Printf("Warning: Learning proposal failed: %v\n", err)
		}
	}
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
	fmt.Println(`Commands:
  /exit, /quit, /q    - Exit the CLI
  /clear              - Clear conversation history
  /history            - Show conversation history
  /save               - Save current session
  /load [id]          - Load session (or last if no ID)
  /sessions           - List recent sessions
  /undo [all]         - Undo last file change (restore from .bak) or undo all changes
  /undo history       - Show past session changes
  /undo session <id>  - Undo all changes from specific session
  /changes            - Show file change history with undo status
  /stats              - Show session statistics (time, messages, tokens, cost)
  /tokens             - Show token usage and context window status
  /copy [code] [-n N] - Copy last AI output to clipboard (code=code blocks only, -n=N-th last output)
  /compress [N] [-c]  - Compress history (keep recent N, -c: use OpenAI Compact API)
  /use <provider> [model] - Switch provider and optionally model (e.g., /use gemini gemini-2.0-flash-exp)
  /providers          - List available providers and their API key status
  /config             - Show/change configuration
                        /config show - Show all settings with diff from defaults
                        /config model <name> - Change default model
  /model [name]       - Show current model or switch model without restart
  /init               - Generate XELYON.md (project config) from codebase analysis
  /sync               - Sync XELYON.md with current codebase (detect new/deleted files, tech changes)
  /paste, /p          - Paste mode for long text (end with empty line x2, END, or Ctrl+D)
  /plan [on|off]      - Toggle Plan Mode (investigation → plan → approval → execution)
  /think [on|off|level] - Toggle Extended Thinking mode (level: low/medium/high/xhigh)
  /lsp [status]       - Show LSP server status (running/not started/disabled)
  /version            - Show version information
  /help               - Show this help

Available tools (AI will use automatically):
  bash        - Execute shell commands
  read_file   - Read file contents
  write_file  - Write/create files (creates .bak backup)
  str_replace - Replace text in file (creates .bak backup)
  list_dir    - List directory contents
  git_*       - Git operations (status, diff, add, commit, push, log)
  search_code - Search in code files
  search_file - Search for files by name

Tips:
  - Just describe what you want in natural language
  - AI will ask confirmation for dangerous operations
  - Use Ctrl+C to cancel current operation
  - Use /undo to revert file changes (up to 10 recent changes)`)
}
