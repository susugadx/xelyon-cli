package agent

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/version"
)

// handleSpecialCommand は特殊コマンドを処理
func handleSpecialCommand(input string, agent *Agent) bool {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false
	}

	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "/save":
		return handleSaveCommand(agent)
	case "/load":
		return handleLoadCommand(agent, args)
	case "/sessions":
		return handleSessionsCommand(agent)
	case "/undo":
		return handleUndoCommand(agent)
	case "/config":
		return handleConfigCommand(args)
	case "/stats":
		return handleStatsCommand(agent)
	case "/copy":
		return handleCopyCommand(agent, args)
	case "/exit", "/quit", "/q":
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
	case "/repomap":
		return handleRepoMapCommand()
	}
	return false
}

// handleSaveCommand はセッション保存を処理
func handleSaveCommand(agent *Agent) bool {
	if agent.storage == nil {
		red.Println("History storage not available")
		return true
	}

	if err := agent.storage.Save(agent.session); err != nil {
		red.Printf("Failed to save session: %v\n", err)
		return true
	}

	green.Printf("💾 Session saved: %s\n", agent.session.ID)
	return true
}

// handleLoadCommand はセッション読み込みを処理
func handleLoadCommand(agent *Agent, args []string) bool {
	if agent.storage == nil {
		red.Println("History storage not available")
		return true
	}

	sessionID := ""
	if len(args) > 0 {
		sessionID = args[0]
	} else {
		lastID, err := agent.storage.GetLastSession()
		if err != nil {
			red.Printf("No sessions found: %v\n", err)
			return true
		}
		sessionID = lastID
	}

	session, err := agent.storage.Load(sessionID)
	if err != nil {
		red.Printf("Failed to load session: %v\n", err)
		return true
	}

	// セッション置き換え
	agent.session = session
	agent.History = session.ToAPIMessages()

	green.Printf("📂 Loaded session %s (%d messages)\n", sessionID, len(session.Messages))
	return true
}

// handleSessionsCommand はセッション一覧を表示
func handleSessionsCommand(agent *Agent) bool {
	if agent.storage == nil {
		red.Println("History storage not available")
		return true
	}

	sessions, err := agent.storage.ListSessions()
	if err != nil {
		red.Printf("Failed to list sessions: %v\n", err)
		return true
	}

	if len(sessions) == 0 {
		yellow.Println("No sessions found")
		return true
	}

	cyan.Println("\n📚 Recent Sessions:")
	for i, s := range sessions {
		if i >= 10 {
			break
		}

		timeStr := s.LastModified.Format("2006-01-02 15:04")
		preview := s.Preview
		if len(preview) > 60 {
			preview = preview[:60] + "..."
		}

		fmt.Printf("  [%s] %s - %s (%d msgs)\n",
			s.ID, timeStr, preview, s.MessageCount)
	}
	fmt.Println()
	return true
}

// handleUndoCommand は直前のファイル変更を取り消す
func handleUndoCommand(agent *Agent) bool {
	if len(agent.changeStack) == 0 {
		yellow.Println("No changes to undo")
		return true
	}

	// 最後の変更を取得
	lastChange := agent.changeStack[len(agent.changeStack)-1]

	// バックアップが存在しない場合
	if lastChange.BackupPath == "" {
		red.Println("No backup available for last change")
		return true
	}

	// バックアップファイルを確認
	if _, err := os.Stat(lastChange.BackupPath); os.IsNotExist(err) {
		red.Printf("Backup file not found: %s\n", lastChange.BackupPath)
		return true
	}

	// 確認プロンプト
	yellow.Printf("Undo last change?\n")
	fmt.Printf("  File: %s\n", lastChange.FilePath)
	fmt.Printf("  Tool: %s\n", lastChange.Tool)
	fmt.Printf("  Time: %s\n", lastChange.Timestamp.Format("2006-01-02 15:04:05"))
	yellow.Print("Continue? (y/n): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		red.Printf("Failed to read input: %v\n", err)
		return true
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input != "y" && input != "yes" {
		yellow.Println("Undo cancelled")
		return true
	}

	// バックアップから復元
	backupContent, err := os.ReadFile(lastChange.BackupPath)
	if err != nil {
		red.Printf("Failed to read backup: %v\n", err)
		return true
	}

	if err := os.WriteFile(lastChange.FilePath, backupContent, 0644); err != nil {
		red.Printf("Failed to restore file: %v\n", err)
		return true
	}

	// スタックから削除
	agent.changeStack = agent.changeStack[:len(agent.changeStack)-1]

	green.Printf("✅ Undone: %s\n", lastChange.Description)
	green.Printf("   Restored from: %s\n", lastChange.BackupPath)
	return true
}

// handleModelCommand はモデルの表示・切り替えを処理
func handleModelCommand(agent *Agent, args []string) bool {
	// 引数なし → 現在のモデルとプロバイダーを表示
	if len(args) == 0 {
		fmt.Printf("🤖 Current model: %s\n", agent.CurrentModel)
		fmt.Printf("🌐 Provider: %s\n", agent.ProviderName)
		yellow.Println("\nUsage: /model <model-name>")
		yellow.Println("Enter any model name supported by your provider.")

		// Ollamaの場合だけインストール済みモデルを表示
		if agent.ProviderName == "ollama" {
			if ollamaProvider, ok := agent.CurrentProvider.(*api.OllamaProvider); ok {
				models, err := ollamaProvider.ListModels()
				if err != nil {
					yellow.Printf("\nWarning: Could not list Ollama models: %v\n", err)
				} else if len(models) > 0 {
					yellow.Println("\nInstalled Ollama models:")
					for _, model := range models {
						fmt.Printf("  - %s\n", model)
					}
				}
			}
		}
		return true
	}

	// /model <model-name> → モデル切り替え
	newModel := args[0]

	// モデルを切り替え
	oldModel := agent.CurrentModel
	agent.CurrentModel = newModel

	green.Printf("✅ Model switched: %s → %s\n", oldModel, newModel)

	// 設定ファイルにも保存
	cfg, err := config.LoadConfig()
	if err != nil {
		yellow.Printf("Warning: Failed to load config: %v\n", err)
		return true
	}

	cfg.DefaultModel = newModel
	if err := config.SaveConfig(cfg); err != nil {
		yellow.Printf("Warning: Failed to save config: %v\n", err)
		yellow.Println("Model switched for this session only")
		return true
	}

	green.Println("💾 Default model saved to config")
	return true
}

// handleConfigCommand は設定の表示・変更を処理
func handleConfigCommand(args []string) bool {
	cfg, err := config.LoadConfig()
	if err != nil {
		red.Printf("Failed to load config: %v\n", err)
		return true
	}

	// 引数なし → 現在の設定を表示
	if len(args) == 0 {
		cyan.Println("⚙️  Current Configuration:")
		fmt.Printf("  default_model: %s\n", cfg.DefaultModel)
		fmt.Printf("  default_provider: %s\n", cfg.DefaultProvider)
		yellow.Println("\nUsage: /config model <model-name>")
		yellow.Println("Enter any model name supported by your provider.")
		return true
	}

	// /config model <model-name> → モデル変更
	if len(args) >= 2 && args[0] == "model" {
		newModel := args[1]

		// 設定更新（バリデーションなし、任意のモデル名を受け付ける）
		cfg.DefaultModel = newModel
		if err := config.SaveConfig(cfg); err != nil {
			red.Printf("Failed to save config: %v\n", err)
			return true
		}

		green.Printf("✅ Default model updated to: %s\n", newModel)
		yellow.Println("Restart CLI for changes to take effect")
		return true
	}

	yellow.Println("Usage: /config [model <model-name>]")
	return true
}

// handleStatsCommand はセッション統計情報を表示
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

// handleRepoMapCommand はRepo Mapを表示
func handleRepoMapCommand() bool {
	cwd, err := os.Getwd()
	if err != nil {
		yellow.Printf("Warning: Could not get current directory: %v\n", err)
		cwd = "." // フォールバック
	}
	rm := repomap.NewRepoMap(cwd, 0) // 制限なし
	if err := rm.Build(); err != nil {
		red.Printf("Failed to build repo map: %v\n", err)
		return true
	}

	if rm.GetSymbolCount() == 0 {
		yellow.Println("No symbols found in current directory")
		return true
	}

	cyan.Printf("🗺️  Repository Map (%d symbols from %d files)\n\n",
		rm.GetSymbolCount(), len(rm.Files))
	fmt.Println(rm.Generate())
	return true
}

// printHelp はヘルプを表示
func printHelp() {
	fmt.Println(`Commands:
  /exit, /quit, /q  - Exit the CLI
  /clear            - Clear conversation history
  /history          - Show conversation history
  /save             - Save current session
  /load [id]        - Load session (or last if no ID)
  /sessions         - List recent sessions
  /undo             - Undo last file change (restore from .bak)
  /stats            - Show session statistics (time, messages, tokens, cost)
  /copy [code] [-n N] - Copy last AI output to clipboard (code=code blocks only, -n=N-th last output)
  /config           - Show/change configuration (e.g., /config model deepseek-coder)
  /model [name]     - Show current model or switch model without restart
  /repomap          - Show repository code structure map
  /version          - Show version information
  /help             - Show this help

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
