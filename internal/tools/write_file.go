package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// executeWriteFile はファイルに書き込む
func executeWriteFile(path string, content string) (string, string, error) {
	if path == "" {
		return "Error: path is empty", "", nil
	}

	// パストラバーサル防止
	absPath, err := ValidatePath(path)
	if err != nil {
		red.Printf("🚫 Security: %v\n", err)
		return fmt.Sprintf("Error: %v", err), "", nil
	}

	// ファイルが存在するか確認
	exists := false
	if _, err := os.Stat(absPath); err == nil {
		exists = true
	}

	// 確認UI - 変更サマリーを明確に表示
	newLines := strings.Split(content, "\n")

	cyan.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if exists {
		cyan.Printf("📝 write_file (overwrite): %s\n", path)
	} else {
		cyan.Printf("📝 write_file (create): %s\n", path)
	}
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 変更サマリー
	yellow.Println("\n📊 Summary / 変更サマリー:")
	if exists {
		oldContent, _ := os.ReadFile(absPath)
		oldLines := strings.Split(string(oldContent), "\n")
		lineDiff := len(newLines) - len(oldLines)
		fmt.Printf("   • Before: %d lines / 変更前: %d行\n", len(oldLines), len(oldLines))
		fmt.Printf("   • After: %d lines / 変更後: %d行\n", len(newLines), len(newLines))
		if lineDiff > 0 {
			green.Printf("   • Net: +%d lines\n", lineDiff)
		} else if lineDiff < 0 {
			red.Printf("   • Net: %d lines\n", lineDiff)
		} else {
			fmt.Printf("   • Net: 0 lines (same size)\n")
		}
		showDiff(string(oldContent), content, path)
	} else {
		fmt.Printf("   • New file: %d lines / 新規: %d行\n", len(newLines), len(newLines))
		fmt.Printf("   • Size: %d bytes\n", len(content))
		showPreview(content)
	}

	if !confirmWithAutoApprove("write_file", "Create/overwrite this file? / このファイルを作成・上書きしますか？") {
		return "Cancelled by user", "", nil
	}

	// バックアップ作成（既存ファイルの場合のみ）
	backupPath, err := createBackup(absPath)
	if err != nil {
		return fmt.Sprintf("Warning: failed to create backup: %v (continuing anyway)", err), "", nil
	}

	// ディレクトリ作成
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Sprintf("Error creating directory: %v", err), "", nil
	}

	// 書き込み
	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return fmt.Sprintf("Error writing file: %v", err), "", nil
	}

	green.Printf("✅ Written: %s\n", path)
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), backupPath, nil
}
