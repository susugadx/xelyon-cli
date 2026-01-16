package agent

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

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

	if !promptConfirm("Continue? (y/n): ") {
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
	if !promptConfirm("\nContinue? (y/n): ") {
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
	if !promptConfirm("\nContinue? (y/n): ") {
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
