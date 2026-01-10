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

	// 確認UI
	lines := strings.Split(content, "\n")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if exists {
		cyan.Printf("📝 Create/Overwrite File / ファイルの上書き\n")
	} else {
		cyan.Printf("📝 Create File / ファイルの新規作成\n")
	}
	cyan.Printf("📂 Path / パス: %s\n", path)
	cyan.Printf("📏 Size / サイズ: %d lines / 行\n", len(lines))
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// diff表示（既存ファイルの場合）
	if exists {
		oldContent, _ := os.ReadFile(absPath)
		showDiff(string(oldContent), content, path)
	} else {
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
