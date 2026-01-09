package tools

import (
	"fmt"
	"os"
	"strings"
)

// executeAppendFile はファイル末尾にコンテンツを追加
func executeAppendFile(path, content string) (string, string, error) {
	// 引数検証
	if path == "" {
		return "Error: path is empty", "", nil
	}
	if content == "" {
		return "Error: content is empty", "", nil
	}

	// パストラバーサル防止
	absPath, err := ValidatePath(path)
	if err != nil {
		red.Printf("🚫 Security: %v\n", err)
		return fmt.Sprintf("Error: %v", err), "", nil
	}

	// ファイルが存在するかチェック
	exists := false
	if _, err := os.Stat(absPath); err == nil {
		exists = true
	}

	// バックアップ作成（既存ファイルのみ）
	backupPath, err := createBackup(absPath)
	if err != nil {
		return fmt.Sprintf("Error creating backup: %v", err), "", nil
	}

	// プレビュー表示（確認なし、非破壊的操作）
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if exists {
		cyan.Printf("➕ Append to File / ファイルに追記\n")
	} else {
		cyan.Printf("➕ Create File with Content / ファイルを作成\n")
	}
	cyan.Printf("📂 Path / パス: %s\n", path)
	contentLines := strings.Split(content, "\n")
	cyan.Printf("📏 Adding / 追加: %d lines / 行\n", len(contentLines))
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 既存ファイルの最後の部分を表示
	if exists {
		oldContent, _ := os.ReadFile(absPath)
		if len(oldContent) > 0 {
			oldLines := strings.Split(string(oldContent), "\n")
			yellow.Println("\nExisting file (last 10 lines) / 既存ファイル（最終10行）:")
			cyan.Println("┌" + strings.Repeat("─", 60) + "┐")
			startLine := len(oldLines) - 10
			if startLine < 0 {
				startLine = 0
			}
			for i := startLine; i < len(oldLines) && i < startLine+10; i++ {
				fmt.Printf("│ %s\n", oldLines[i])
			}
			cyan.Println("└" + strings.Repeat("─", 60) + "┘")
		}
	}

	// 追加するコンテンツをプレビュー
	yellow.Println("\nContent to append / 追記する内容:")
	cyan.Println("┌" + strings.Repeat("─", 60) + "┐")
	for i, line := range contentLines {
		if i >= 10 {
			yellow.Printf("│ ... (%d more lines / 行省略)\n", len(contentLines)-10)
			break
		}
		green.Printf("│ + %s\n", line)
	}
	cyan.Println("└" + strings.Repeat("─", 60) + "┘")
	fmt.Println()

	// ファイルを開く（追記モード）
	file, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Sprintf("Error opening file: %v", err), "", nil
	}
	defer file.Close()

	// コンテンツを追記
	if _, err := file.WriteString(content); err != nil {
		return fmt.Sprintf("Error appending to file: %v", err), "", nil
	}

	green.Printf("✅ Appended: %s\n", path)
	return fmt.Sprintf("Successfully appended %d bytes to %s", len(content), path), backupPath, nil
}
