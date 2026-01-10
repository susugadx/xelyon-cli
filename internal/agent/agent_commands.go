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
	"github.com/susugadx/xelyon-cli/internal/memory"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools"
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
	case "/memory":
		return handleMemoryCommand(args)
	case "/plan":
		return handlePlanCommand(agent, args)
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

// handleUndoCommand は直前のファイル変更を取り消す（またはすべて取り消す）
func handleUndoCommand(agent *Agent, args []string) bool {
	// /undo history → 過去セッションの変更履歴を表示
	if len(args) > 0 && args[0] == "history" {
		return handleUndoHistory(agent)
	}

	// /undo session <session_id> → 指定セッションの変更を取り消す
	if len(args) > 0 && args[0] == "session" {
		if len(args) < 2 {
			red.Println("Usage: /undo session <session_id>")
			return true
		}
		return handleUndoSession(agent, args[1])
	}

	if len(agent.changeStack) == 0 {
		yellow.Println("No changes to undo in current session")
		yellow.Println("Hint: Use /undo history to see past sessions")
		return true
	}

	// /undo all の場合
	if len(args) > 0 && args[0] == "all" {
		return handleUndoAll(agent)
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

// handleUndoAll はすべてのファイル変更を取り消す
func handleUndoAll(agent *Agent) bool {
	totalChanges := len(agent.changeStack)

	// 確認プロンプト
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("⚠️  Undo All Changes / すべての変更を取り消し\n")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("取り消す変更数: %d 件\n", totalChanges)
	yellow.Println("\n⚠️  Warning: すべてのファイルがバックアップから復元されます")

	// 確認
	fmt.Printf("\nContinue? (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		red.Printf("Failed to read input: %v\n", err)
		return true
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input != "y" && input != "yes" {
		yellow.Println("Cancelled")
		return true
	}

	// 逆順で処理（新しい変更から古い変更へ）
	successCount := 0
	failCount := 0

	fmt.Println()
	cyan.Println("Restoring files...")

	for i := len(agent.changeStack) - 1; i >= 0; i-- {
		change := agent.changeStack[i]

		// バックアップがない場合
		if change.BackupPath == "" {
			yellow.Printf("  ⚠️  [%d/%d] %s - バックアップなし\n", totalChanges-i, totalChanges, change.FilePath)
			failCount++
			continue
		}

		// バックアップファイルを確認
		if _, err := os.Stat(change.BackupPath); os.IsNotExist(err) {
			yellow.Printf("  ⚠️  [%d/%d] %s - バックアップファイルが見つかりません\n", totalChanges-i, totalChanges, change.FilePath)
			failCount++
			continue
		}

		// バックアップから復元
		backupContent, err := os.ReadFile(change.BackupPath)
		if err != nil {
			red.Printf("  ❌ [%d/%d] %s - バックアップ読み込み失敗: %v\n", totalChanges-i, totalChanges, change.FilePath, err)
			failCount++
			continue
		}

		if err := os.WriteFile(change.FilePath, backupContent, 0644); err != nil {
			red.Printf("  ❌ [%d/%d] %s - 復元失敗: %v\n", totalChanges-i, totalChanges, change.FilePath, err)
			failCount++
			continue
		}

		green.Printf("  ✅ [%d/%d] %s\n", totalChanges-i, totalChanges, change.FilePath)
		successCount++
	}

	// スタックをクリア
	agent.changeStack = []tools.FileChange{}

	// 結果表示
	fmt.Println()
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	green.Printf("✅ 成功: %d 件\n", successCount)
	if failCount > 0 {
		yellow.Printf("⚠️  失敗/スキップ: %d 件\n", failCount)
	}
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return true
}

// handleUndoHistory は過去セッションの変更履歴を表示
func handleUndoHistory(agent *Agent) bool {
	if agent.changeStorage == nil {
		red.Println("Change storage not available")
		return true
	}

	sessions, err := agent.changeStorage.ListSessions()
	if err != nil {
		red.Printf("Failed to list sessions: %v\n", err)
		return true
	}

	if len(sessions) == 0 {
		yellow.Println("No past session changes found")
		return true
	}

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("📜 Past Session Changes / 過去セッションの変更履歴\n")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// 最新10セッションのみ表示
	displayCount := 10
	if len(sessions) < displayCount {
		displayCount = len(sessions)
	}

	for i := 0; i < displayCount; i++ {
		session := sessions[i]

		// タイムスタンプ
		timeStr := session.LastChange.Format("2006-01-02 15:04")

		// セッション情報表示
		fmt.Printf("  [%s] %s\n", session.SessionID, timeStr)
		fmt.Printf("      Changes: %d | Files: %d\n", session.ChangeCount, len(session.FilesChanged))

		// 変更されたファイル一覧（最大5件）
		fileCount := 0
		for filePath, count := range session.FilesChanged {
			if fileCount >= 5 {
				remaining := len(session.FilesChanged) - 5
				fmt.Printf("      ... and %d more files\n", remaining)
				break
			}
			fmt.Printf("      - %s (%d changes)\n", filePath, count)
			fileCount++
		}

		if i < displayCount-1 {
			fmt.Println()
		}
	}

	fmt.Println()
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	yellow.Println("使い方:")
	yellow.Println("  /undo session <session_id>  - セッションの変更を取り消し")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return true
}

// handleUndoSession は指定セッションの変更を取り消す
func handleUndoSession(agent *Agent, sessionID string) bool {
	if agent.changeStorage == nil {
		red.Println("Change storage not available")
		return true
	}

	// セッションの変更履歴を読み込み
	changes, err := agent.changeStorage.LoadSessionChanges(sessionID)
	if err != nil {
		red.Printf("Failed to load session changes: %v\n", err)
		return true
	}

	if len(changes) == 0 {
		yellow.Printf("No changes found for session: %s\n", sessionID)
		return true
	}

	// 確認プロンプト
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("⚠️  Undo Session Changes / セッションの変更を取り消し\n")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Session ID: %s\n", sessionID)
	fmt.Printf("取り消す変更数: %d 件\n", len(changes))
	yellow.Println("\n⚠️  Warning: すべての変更がバックアップから復元されます")

	// 確認
	fmt.Print("\nContinue? (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		red.Printf("Failed to read input: %v\n", err)
		return true
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input != "y" && input != "yes" {
		yellow.Println("Cancelled")
		return true
	}

	// 逆順で処理（新しい変更から古い変更へ）
	successCount := 0
	failCount := 0

	fmt.Println()
	cyan.Println("Restoring files...")

	for i := len(changes) - 1; i >= 0; i-- {
		change := changes[i]

		// バックアップから復元
		if err := agent.changeStorage.UndoSessionChange(change); err != nil {
			red.Printf("  ❌ [%d/%d] %s - %v\n", len(changes)-i, len(changes), change.FilePath, err)
			failCount++
			continue
		}

		green.Printf("  ✅ [%d/%d] %s\n", len(changes)-i, len(changes), change.FilePath)
		successCount++
	}

	// 結果表示
	fmt.Println()
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	green.Printf("✅ 成功: %d 件\n", successCount)
	if failCount > 0 {
		yellow.Printf("⚠️  失敗/スキップ: %d 件\n", failCount)
	}
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return true
}

// handleChangesCommand は変更履歴を表示
func handleChangesCommand(agent *Agent) bool {
	if len(agent.changeStack) == 0 {
		yellow.Println("変更履歴はありません")
		return true
	}

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("📝 Change History / 変更履歴 (%d 件)\n", len(agent.changeStack))
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	for i, change := range agent.changeStack {
		// 変更種別を日本語で表示
		changeType := getChangeTypeJP(change.Tool)

		// タイムスタンプ
		timeStr := change.Timestamp.Format("15:04:05")

		// Undo可能かチェック（バックアップファイルの存在確認）
		canUndo := "❌"
		if change.BackupPath != "" {
			if _, err := os.Stat(change.BackupPath); err == nil {
				canUndo = "✅"
			}
		}

		// 表示
		fmt.Printf("  [%d] %s %s\n", i+1, changeType, change.FilePath)
		fmt.Printf("      時刻: %s | ツール: %s | Undo: %s\n", timeStr, change.Tool, canUndo)

		// 説明がある場合は表示
		if change.Description != "" {
			fmt.Printf("      説明: %s\n", change.Description)
		}

		fmt.Println()
	}

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	yellow.Println("使い方:")
	yellow.Println("  /undo           - 最後の変更を取り消し")
	yellow.Println("  /undo all       - すべての変更を取り消し")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return true
}

// getChangeTypeJP は変更種別を日本語で返す
func getChangeTypeJP(tool string) string {
	switch tool {
	case "write_file":
		return "📝 作成"
	case "str_replace", "append_file", "prepend_file", "insert_after", "insert_before":
		return "✏️  編集"
	case "delete_file":
		return "🗑️  削除"
	case "move_file":
		return "📦 移動"
	case "copy_file":
		return "📋 コピー"
	case "delete_lines":
		return "✂️  行削除"
	default:
		return "🔧 変更"
	}
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
	fmt.Print("\nContinue? (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		red.Printf("Failed to read input: %v\n", err)
		return true
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input != "y" && input != "yes" {
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

// handleUseCommand はプロバイダーを切り替える
func handleUseCommand(agent *Agent, args []string) bool {
	if len(args) == 0 {
		yellow.Println("Usage: /use <provider>")
		yellow.Println("Available providers: deepseek, claude, openai, gemini, groq, ollama")
		return true
	}

	providerName := args[0]

	// サポートされているプロバイダーかチェック
	validProviders := map[string]bool{
		"deepseek": true,
		"claude":   true,
		"openai":   true,
		"gemini":   true,
		"groq":     true,
		"ollama":   true,
	}

	if !validProviders[providerName] {
		red.Printf("Unknown provider: %s\n", providerName)
		yellow.Println("Available providers: deepseek, claude, openai, gemini, groq, ollama")
		return true
	}

	// 既に同じプロバイダーの場合
	if agent.ProviderName == providerName {
		yellow.Printf("Already using %s\n", providerName)
		return true
	}

	// プロバイダー切り替え実行
	if err := agent.SwitchProvider(providerName); err != nil {
		red.Printf("❌ %v\n", err)

		// API キー設定方法を表示
		switch providerName {
		case "deepseek":
			yellow.Println("\n設定方法:")
			yellow.Println("  export DEEPSEEK_API_KEY=your-api-key")
		case "openai":
			yellow.Println("\n設定方法:")
			yellow.Println("  export OPENAI_API_KEY=your-api-key")
		case "claude":
			yellow.Println("\n設定方法:")
			yellow.Println("  export ANTHROPIC_API_KEY=your-api-key")
		case "gemini":
			yellow.Println("\n設定方法:")
			yellow.Println("  export GEMINI_API_KEY=your-api-key")
		case "groq":
			yellow.Println("\n設定方法:")
			yellow.Println("  export GROQ_API_KEY=your-api-key")
		}
		return true
	}

	return true
}

// handleProvidersCommand は利用可能なプロバイダー一覧を表示
func handleProvidersCommand(agent *Agent) bool {
	providers := []string{"deepseek", "claude", "openai", "gemini", "groq", "ollama"}

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Println("📡 利用可能なプロバイダー / Available Providers")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	for _, provider := range providers {
		// 現在使用中かチェック
		isCurrent := agent.ProviderName == provider
		hasAPIKey := IsAPIKeyAvailable(provider)

		// アイコン
		icon := "  "
		if isCurrent {
			icon = "✓ "
		}

		// ステータス
		status := ""
		if provider == "ollama" {
			status = "(ローカル)"
		} else if hasAPIKey {
			status = "(API key設定済み)"
		} else {
			status = "(API key未設定)"
		}

		// 色付け
		if isCurrent {
			green.Printf("%s%-12s %s\n", icon, provider, status)
		} else if hasAPIKey {
			fmt.Printf("%s%-12s %s\n", icon, provider, status)
		} else {
			// API key未設定は薄く表示
			fmt.Printf("%s%-12s %s\n", icon, provider, status)
		}
	}

	fmt.Println()
	cyan.Println("使い方: /use <provider>")
	cyan.Println("例: /use claude")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return true
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

// handleMemoryCommand は記憶の管理を処理
func handleMemoryCommand(args []string) bool {
	store, err := memory.NewMemoryStore()
	if err != nil {
		red.Printf("Failed to initialize memory store: %v\n", err)
		return true
	}

	// 引数なし、または "list" → 一覧表示
	if len(args) == 0 || (len(args) == 1 && args[0] == "list") {
		memories := store.List()
		if len(memories) == 0 {
			yellow.Println("No memories stored")
			yellow.Println("\nUsage: /memory <text>  または  /memory add <text>")
			return true
		}

		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("🧠 Memories / 記憶 (%d 件)\n", len(memories))
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		for i, m := range memories {
			// プロジェクト別かグローバルかを表示
			scope := "Global"
			if m.Project != "" {
				scope = fmt.Sprintf("Project: %s", m.Project)
			}

			// タイムスタンプ
			timeStr := m.CreatedAt.Format("2006-01-02 15:04")

			fmt.Printf("  [%s] %s\n", m.ID, m.Content)
			fmt.Printf("      %s | %s\n", scope, timeStr)

			if i < len(memories)-1 {
				fmt.Println()
			}
		}

		fmt.Println()
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		yellow.Println("使い方:")
		yellow.Println("  /memory <text>         - 記憶を追加（グローバル）")
		yellow.Println("  /memory add <text>     - 記憶を追加（グローバル）")
		yellow.Println("  /memory project <text> - 記憶を追加（プロジェクト別）")
		yellow.Println("  /memory list           - 記憶一覧")
		yellow.Println("  /memory delete <id>    - 記憶削除")
		yellow.Println("  /memory clear          - 全記憶削除")
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		return true
	}

	subcommand := args[0]

	// /memory add <text> → グローバル記憶追加
	if subcommand == "add" {
		if len(args) < 2 {
			red.Println("Usage: /memory add <text>")
			return true
		}

		content := strings.Join(args[1:], " ")
		m, err := store.Add(content, false)
		if err != nil {
			red.Printf("Failed to add memory: %v\n", err)
			return true
		}

		green.Printf("✅ Memory added (ID: %s)\n", m.ID)
		green.Printf("   Scope: Global\n")
		green.Printf("   Content: %s\n", m.Content)
		return true
	}

	// /memory project <text> → プロジェクト別記憶追加
	if subcommand == "project" {
		if len(args) < 2 {
			red.Println("Usage: /memory project <text>")
			return true
		}

		if store.ProjectPath == "" {
			red.Println("Not in a project directory (.xelyon not found)")
			yellow.Println("Hint: Create .xelyon directory in project root, or use /memory add for global memory")
			return true
		}

		content := strings.Join(args[1:], " ")
		m, err := store.Add(content, true)
		if err != nil {
			red.Printf("Failed to add memory: %v\n", err)
			return true
		}

		green.Printf("✅ Memory added (ID: %s)\n", m.ID)
		green.Printf("   Scope: Project (%s)\n", m.Project)
		green.Printf("   Content: %s\n", m.Content)
		return true
	}

	// /memory delete <id> → 記憶削除
	if subcommand == "delete" {
		if len(args) < 2 {
			red.Println("Usage: /memory delete <id>")
			return true
		}

		id := args[1]
		if err := store.Delete(id); err != nil {
			red.Printf("Failed to delete memory: %v\n", err)
			return true
		}

		green.Printf("✅ Memory deleted (ID: %s)\n", id)
		return true
	}

	// /memory clear → 全記憶削除
	if subcommand == "clear" {
		// 確認プロンプト
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("⚠️  Clear All Memories / 全記憶削除\n")
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("削除する記憶数: %d 件\n", len(store.List()))
		yellow.Println("\n⚠️  Warning: すべての記憶が削除されます（復元不可）")

		// 確認
		fmt.Print("\nContinue? (y/n): ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			red.Printf("Failed to read input: %v\n", err)
			return true
		}

		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			yellow.Println("Cancelled")
			return true
		}

		// 全削除実行
		if err := store.Clear(false); err != nil {
			red.Printf("Failed to clear memories: %v\n", err)
			return true
		}

		green.Println("✅ All memories cleared")
		return true
	}

	// /memory <text> → グローバル記憶追加（ショートカット）
	content := strings.Join(args, " ")
	m, err := store.Add(content, false)
	if err != nil {
		red.Printf("Failed to add memory: %v\n", err)
		return true
	}

	green.Printf("✅ Memory added (ID: %s)\n", m.ID)
	green.Printf("   Scope: Global\n")
	green.Printf("   Content: %s\n", m.Content)
	return true
}

// handlePlanCommand はPlan Modeの切り替えを処理
func handlePlanCommand(agent *Agent, args []string) bool {
	// 引数なし → 現在の状態を表示
	if len(args) == 0 {
		if agent.PlanMode {
			green.Println("📋 Plan Mode: ON")
			yellow.Println("   AI will generate execution plans for your requests.")
			yellow.Println("   Use '/plan off' to disable.")
		} else {
			yellow.Println("📋 Plan Mode: OFF")
			yellow.Println("   AI will execute tasks directly without planning.")
			yellow.Println("   Use '/plan on' to enable.")
		}
		return true
	}

	// /plan on → 有効化
	if args[0] == "on" {
		if agent.PlanMode {
			yellow.Println("Plan Mode is already enabled")
			return true
		}
		agent.PlanMode = true
		green.Println("✅ Plan Mode enabled")
		cyan.Println("   AI will now generate execution plans for your requests.")
		cyan.Println("   You can review and approve each plan before execution.")
		return true
	}

	// /plan off → 無効化
	if args[0] == "off" {
		if !agent.PlanMode {
			yellow.Println("Plan Mode is already disabled")
			return true
		}
		agent.PlanMode = false
		green.Println("✅ Plan Mode disabled")
		cyan.Println("   AI will now execute tasks directly without planning.")
		return true
	}

	// 不正な引数
	yellow.Println("Usage: /plan [on|off]")
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
  /memory [cmd]       - Manage persistent memories across sessions
                        /memory <text> - Add global memory
                        /memory list - List all memories
                        /memory delete <id> - Delete memory
                        /memory clear - Clear all memories
  /stats              - Show session statistics (time, messages, tokens, cost)
  /copy [code] [-n N] - Copy last AI output to clipboard (code=code blocks only, -n=N-th last output)
  /compress [N]       - Compress history (keep recent N messages, default: 10)
  /use <provider>     - Switch provider (deepseek, claude, openai, gemini, groq, ollama)
  /providers          - List available providers and their API key status
  /config             - Show/change configuration (e.g., /config model deepseek-coder)
  /model [name]       - Show current model or switch model without restart
  /plan [on|off]      - Toggle Plan Mode (autonomous execution with planning)
  /repomap            - Show repository code structure map
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
