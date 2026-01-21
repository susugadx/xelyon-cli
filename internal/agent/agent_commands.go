package agent

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/version"
)

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
	parts := strings.Fields(input)
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
	case "/clear":
		agent.History = []api.Message{}
		green.Println("🗑️  History cleared")
		return true
	case "/history":
		fmt.Printf("📜 %d messages in history\n", len(agent.History))
		for i, msg := range agent.History {
			role := "👤"
			if msg.Role == "assistant" {
				role = "🤖"
			}
			preview := msg.Content
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			fmt.Printf("  %d. %s %s\n", i+1, role, preview)
		}
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
	}
	return false
}

// handleStatsCommand はセッション統計を表示
func handleStatsCommand(agent *Agent) bool {
	if agent.Stats == nil {
		yellow.Println("Statistics not available")
		return true
	}

	stats := agent.Stats

	// セッションファイルパスとサイズを取得
	sessionPath := ""
	sessionSize := int64(0)
	if agent.session != nil {
		sessionPath = fmt.Sprintf("~/.xelyon/sessions/%s.json", agent.session.ID)
		if agent.storage != nil {
			// セッションファイルの実際のパスを構築
			homeDir, err := os.UserHomeDir()
			if err == nil {
				fullPath := fmt.Sprintf("%s/.xelyon/sessions/%s.json", homeDir, agent.session.ID)
				if size, err := GetSessionFileSize(fullPath); err == nil {
					sessionSize = size
				}
			}
		}
	}

	// 統計情報を表示
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("📊 Session Statistics / セッション統計\n")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println()
	green.Println("⏱️  Time / 経過時間")
	fmt.Printf("  Elapsed: %s\n", stats.FormatElapsedTime())

	fmt.Println()
	green.Println("💬 Messages / メッセージ数")
	fmt.Printf("  User:      %d\n", stats.UserMessages)
	fmt.Printf("  Assistant: %d\n", stats.AssistantMessages)
	fmt.Printf("  Total:     %d\n", stats.TotalMessages())

	fmt.Println()
	green.Println("🔧 Tool Executions / ツール実行回数")
	if stats.TotalToolExecutions() > 0 {
		fmt.Printf("  Total: %d\n", stats.TotalToolExecutions())
		fmt.Println("  Breakdown:")
		for tool, count := range stats.ToolExecutions {
			fmt.Printf("    - %-15s: %d\n", tool, count)
		}
	} else {
		fmt.Println("  No tools executed yet")
	}

	fmt.Println()
	green.Println("🤖 Provider / プロバイダー")
	fmt.Printf("  Name: %s\n", stats.Provider)
	fmt.Printf("  Model: %s\n", agent.CurrentModel)

	fmt.Println()
	green.Println("💰 Token Usage & Cost / トークン使用量とコスト")
	if stats.TotalTokens() > 0 {
		fmt.Printf("  Input:  %s tokens\n", formatNumber(stats.InputTokens))
		fmt.Printf("  Output: %s tokens\n", formatNumber(stats.OutputTokens))
		fmt.Printf("  Total:  %s tokens\n", formatNumber(stats.TotalTokens()))
		cost := stats.EstimatedCost()
		if cost > 0 {
			fmt.Printf("  Estimated Cost: $%.4f USD\n", cost)
		} else {
			fmt.Println("  Cost: Free (local model)")
		}
	} else {
		yellow.Println("  No token usage data available")
		yellow.Println("  (Token tracking requires API support)")
	}

	fmt.Println()
	green.Println("📁 Session File / セッションファイル")
	if sessionPath != "" {
		fmt.Printf("  Path: %s\n", sessionPath)
		if sessionSize > 0 {
			fmt.Printf("  Size: %s\n", FormatFileSize(sessionSize))
		}
	} else {
		yellow.Println("  No session file")
	}

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return true
}

// formatNumber はカンマ区切りの数値を返す
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%s,%03d", formatNumber(n/1000), n%1000)
}

// handleCopyCommand は最後のAI出力をクリップボードにコピー
func handleCopyCommand(agent *Agent, args []string) bool {
	if len(agent.lastOutputs) == 0 {
		yellow.Println("No AI output to copy yet")
		return true
	}

	// デフォルト: 最後の出力
	outputIndex := len(agent.lastOutputs) - 1
	codeOnly := false

	// 引数解析
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "code":
			codeOnly = true
		case "-n":
			if i+1 < len(args) {
				n, err := strconv.Atoi(args[i+1])
				if err != nil {
					red.Printf("Invalid number: %s\n", args[i+1])
					return true
				}
				if n < 1 || n > len(agent.lastOutputs) {
					red.Printf("Index out of range (1-%d): %d\n", len(agent.lastOutputs), n)
					return true
				}
				outputIndex = len(agent.lastOutputs) - n
				i++ // skip next arg
			} else {
				red.Println("Missing value for -n flag")
				return true
			}
		default:
			yellow.Printf("Unknown argument: %s\n", arg)
			yellow.Println("Usage: /copy [code] [-n <number>]")
			return true
		}
	}

	output := agent.lastOutputs[outputIndex]

	// コードブロックのみ抽出
	if codeOnly {
		codeBlocks := extractCodeBlocks(output)
		if len(codeBlocks) == 0 {
			yellow.Println("No code blocks found in output")
			return true
		}
		output = strings.Join(codeBlocks, "\n\n")
	}

	// クリップボードにコピー
	if err := clipboard.WriteAll(output); err != nil {
		red.Printf("Failed to copy to clipboard: %v\n", err)
		if strings.Contains(err.Error(), "xclip") || strings.Contains(err.Error(), "xsel") {
			yellow.Println("\nLinux requires xclip or xsel:")
			yellow.Println("  Ubuntu/Debian: sudo apt-get install xclip")
			yellow.Println("  Fedora/RHEL:   sudo dnf install xclip")
			yellow.Println("  Arch:          sudo pacman -S xclip")
		}
		return true
	}

	// 成功メッセージ
	lines := strings.Count(output, "\n") + 1
	chars := len(output)
	green.Printf("✅ Copied to clipboard (%d lines, %d chars", lines, chars)
	if codeOnly {
		fmt.Printf(", code blocks only")
	}
	fmt.Println(")")

	return true
}

// extractCodeBlocks は ```で囲まれたコードブロックを抽出
func extractCodeBlocks(text string) []string {
	// 正規表現: ```language\n...```
	re := regexp.MustCompile("(?s)```\\w*\\n(.*?)```")
	matches := re.FindAllStringSubmatch(text, -1)

	blocks := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			blocks = append(blocks, strings.TrimSpace(match[1]))
		}
	}

	return blocks
}

// handleCompressCommand は会話履歴を圧縮
func handleCompressCommand(agent *Agent, args []string) bool {
	// デフォルト: 最新10件を保持
	keepRecent := 10

	// 引数解析
	if len(args) > 0 {
		n, err := strconv.Atoi(args[0])
		if err != nil {
			red.Printf("Invalid number: %s\n", args[0])
			yellow.Println("Usage: /compress [keep_recent]")
			return true
		}
		if n < 1 {
			red.Println("keep_recent must be at least 1")
			return true
		}
		keepRecent = n
	}

	// 確認プロンプト
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("🗜️  Compress History / 会話履歴を圧縮\n")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("現在の履歴: %d messages\n", len(agent.History))
	fmt.Printf("保持する最新件数: %d messages\n", keepRecent)
	fmt.Printf("圧縮対象: %d messages\n", len(agent.History)-keepRecent)
	yellow.Println("\n⚠️  Warning: 圧縮後、古いメッセージはサマリーに置き換わります")

	// 確認
	if !promptConfirm("\nContinue? (y/n): ") {
		yellow.Println("Cancelled")
		return true
	}

	// 圧縮実行
	if err := agent.CompressHistory(keepRecent); err != nil {
		red.Printf("圧縮に失敗しました: %v\n", err)
		return true
	}

	return true
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
  /copy [code] [-n N] - Copy last AI output to clipboard (code=code blocks only, -n=N-th last output)
  /compress [N]       - Compress history (keep recent N messages, default: 10)
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

// handleLSPCommand は LSP サーバーの状態を表示・管理
func handleLSPCommand(agent *Agent, args []string) bool {
	lspClient := agent.GetLSPClient()
	if lspClient == nil {
		yellow.Println("LSP is not enabled.")
		yellow.Println("To enable, set 'lsp.enabled: true' in ~/.xelyon/config.yaml")
		return true
	}

	// サブコマンド処理（デフォルトはstatus）
	subCmd := "status"
	if len(args) > 0 {
		subCmd = args[0]
	}

	switch subCmd {
	case "status":
		return handleLSPStatus(agent, lspClient)

	case "detect":
		return handleLSPDetect(agent, lspClient)

	case "install":
		return handleLSPInstall(agent, lspClient, args[1:])

	default:
		yellow.Printf("Unknown subcommand: %s\n", subCmd)
		yellow.Println("Usage: /lsp [status|detect|install <language|all>]")
		return true
	}
}

// handleLSPStatus は LSP サーバーの状態を表示
func handleLSPStatus(agent *Agent, lspClient *lsp.Client) bool {
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Println("🔌 LSP Server Status")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	status := lspClient.Status()
	if len(status) == 0 {
		yellow.Println("  No LSP servers configured")
		return true
	}

	// 未インストールサーバーを収集
	var notInstalled []string

	for lang, state := range status {
		var icon string
		switch {
		case state == "running":
			icon = "🟢"
		case state == "disabled":
			icon = "🔴"
		case strings.HasPrefix(state, "not installed"):
			icon = "⚠️ "
			notInstalled = append(notInstalled, lang)
		default:
			icon = "⚪"
		}
		fmt.Printf("  %s %-12s : %s\n", icon, lang, state)
	}

	// プロジェクト内の言語を検出
	cwd, _ := os.Getwd()
	detectedLangs, _ := lsp.DetectProjectLanguages(cwd)
	if len(detectedLangs) > 0 {
		fmt.Println()
		var langNames []string
		for _, info := range detectedLangs {
			langNames = append(langNames, info.ServerKey)
		}
		cyan.Printf("🔍 Detected languages in project: %s\n", strings.Join(langNames, ", "))
	}

	// 未インストールサーバーのインストール提案
	if len(notInstalled) > 0 {
		fmt.Println()
		yellow.Println("💡 Missing LSP servers:")
		for _, lang := range notInstalled {
			if info, ok := lsp.GetInstallInfo(lang); ok {
				fmt.Printf("   %s: %s\n", lang, info.Commands[0])
			}
		}
		fmt.Println()
		yellow.Printf("   Run '/lsp install <language>' to install\n")
		yellow.Printf("   Run '/lsp install all' to install all missing servers\n")
	} else {
		fmt.Println()
		green.Println("Tip: LSP servers start lazily when first used.")
	}

	return true
}

// handleLSPDetect はプロジェクト内の言語を検出して表示
func handleLSPDetect(agent *Agent, lspClient *lsp.Client) bool {
	cyan.Println("🔍 Detecting languages in project...")
	fmt.Println()

	cwd, err := os.Getwd()
	if err != nil {
		red.Printf("Error getting current directory: %v\n", err)
		return true
	}

	languages, err := lsp.DetectProjectLanguages(cwd)
	if err != nil {
		red.Printf("Error detecting languages: %v\n", err)
		return true
	}

	if len(languages) == 0 {
		yellow.Println("No supported languages detected in this project.")
		return true
	}

	cyan.Println("Found:")
	for _, info := range languages {
		exts := strings.Join(info.Extensions, ", ")
		fmt.Printf("  - %s (%s): %d files\n", info.ServerKey, exts, info.FileCount)
	}

	// LSPサーバーの状態を確認
	status := lspClient.Status()
	fmt.Println()
	cyan.Println("LSP Status:")

	var notInstalled []string
	for _, info := range languages {
		serverKey := info.ServerKey
		state, ok := status[serverKey]

		if !ok {
			fmt.Printf("  ❓ %s: not configured\n", serverKey)
			continue
		}

		if strings.HasPrefix(state, "not installed") {
			fmt.Printf("  ❌ %s: not installed\n", serverKey)
			notInstalled = append(notInstalled, serverKey)
		} else if state == "disabled" {
			fmt.Printf("  🔴 %s: disabled\n", serverKey)
		} else {
			installInfo, _ := lsp.GetInstallInfo(serverKey)
			fmt.Printf("  ✅ %s: installed (%s)\n", serverKey, installInfo.PackageName)
		}
	}

	if len(notInstalled) > 0 {
		fmt.Println()
		yellow.Printf("Run '/lsp install %s' to install missing servers.\n", notInstalled[0])
		if len(notInstalled) > 1 {
			yellow.Println("Run '/lsp install all' to install all missing servers.")
		}
	}

	return true
}

// handleLSPInstall は LSP サーバーをインストール
func handleLSPInstall(agent *Agent, lspClient *lsp.Client, args []string) bool {
	if len(args) == 0 {
		yellow.Println("Usage: /lsp install <language|all>")
		yellow.Println("Available languages: go, typescript, python, rust")
		return true
	}

	target := args[0]

	if target == "all" {
		return handleLSPInstallAll(agent, lspClient)
	}

	// 単一言語のインストール
	info, ok := lsp.GetInstallInfo(target)
	if !ok {
		red.Printf("Unknown language: %s\n", target)
		yellow.Println("Available languages: go, typescript, python, rust")
		return true
	}

	// インストール済みチェック
	status := lspClient.Status()
	if state, exists := status[target]; exists && !strings.HasPrefix(state, "not installed") {
		green.Printf("✅ %s is already installed (%s)\n", target, info.PackageName)
		return true
	}

	// インストール実行
	cyan.Printf("📦 Installing %s LSP server (%s)...\n", target, info.PackageName)
	fmt.Println()
	cyan.Printf("Running: %s\n", info.Commands[0])
	fmt.Println()

	if err := lsp.RunInstall(target); err != nil {
		red.Printf("❌ Installation failed: %v\n", err)
		fmt.Println()
		yellow.Println("Try installing manually:")
		for _, cmd := range info.Commands {
			fmt.Printf("   %s\n", cmd)
		}
		return true
	}

	fmt.Println()
	green.Printf("✅ %s installed successfully!\n", info.PackageName)
	fmt.Println()
	yellow.Println("Tip: LSP server will start automatically when you use LSP tools.")
	return true
}

// handleLSPInstallAll は未インストールの全LSPサーバーをインストール
func handleLSPInstallAll(agent *Agent, lspClient *lsp.Client) bool {
	status := lspClient.Status()

	// 未インストールのサーバーを収集
	var toInstall []string
	for lang, state := range status {
		if strings.HasPrefix(state, "not installed") {
			toInstall = append(toInstall, lang)
		}
	}

	if len(toInstall) == 0 {
		green.Println("✅ All configured LSP servers are already installed!")
		return true
	}

	cyan.Println("📦 Installing missing LSP servers...")
	fmt.Println()

	successCount := 0
	for i, lang := range toInstall {
		info, ok := lsp.GetInstallInfo(lang)
		if !ok {
			continue
		}

		fmt.Printf("[%d/%d] Installing %s (%s)...\n", i+1, len(toInstall), lang, info.PackageName)
		fmt.Printf("      %s\n", info.Commands[0])

		if err := lsp.RunInstall(lang); err != nil {
			red.Printf("      ❌ Failed: %v\n", err)
		} else {
			green.Println("      ✅ Success")
			successCount++
		}
		fmt.Println()
	}

	if successCount == len(toInstall) {
		green.Println("All servers installed!")
	} else {
		yellow.Printf("%d of %d servers installed successfully.\n", successCount, len(toInstall))
	}

	return true
}

// handlePlanCommand は Plan Mode の切り替え
func handlePlanCommand(agent *Agent, args []string) bool {
	if len(args) > 0 {
		switch args[0] {
		case "on":
			agent.PlanModeEnabled = true
			green.Println("✅ Plan Mode ON - 調査→計画→承認→実行")
			return true
		case "off":
			agent.PlanModeEnabled = false
			green.Println("✅ Plan Mode OFF - 通常モード")
			return true
		case "status":
			if agent.PlanModeEnabled {
				cyan.Println("📋 Plan Mode: ON")
				fmt.Println("   調査 → 計画 → 承認 → 実行 のフローで処理")
			} else {
				cyan.Println("📋 Plan Mode: OFF")
				fmt.Println("   通常モード（ツール個別確認）")
			}
			return true
		}
	}

	// 引数なし：現在のステータスを表示
	if agent.PlanModeEnabled {
		cyan.Println("📋 Plan Mode: ON")
		fmt.Println("   調査 → 計画 → 承認 → 実行 のフローで処理")
	} else {
		cyan.Println("📋 Plan Mode: OFF")
		fmt.Println("   通常モード（ツール個別確認）")
	}
	return true
}
