package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// executePrependFile はファイル先頭にコンテンツを追加
func executePrependFile(path, content string) (string, string, error) {
	if path == "" {
		return "Error: path is empty", "", nil
	}
	if content == "" {
		return "Error: content is empty", "", nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), "", nil
	}

	exists := false
	var oldContent []byte
	if _, err := os.Stat(absPath); err == nil {
		exists = true
		var readErr error
		oldContent, readErr = os.ReadFile(absPath)
		if readErr != nil && !os.IsNotExist(readErr) {
			// ファイル不存在以外のエラーは報告
			return fmt.Sprintf("Error: Failed to read file: %v", readErr), "", nil
		}
	}

	backupPath, err := createBackup(absPath)
	if err != nil {
		return fmt.Sprintf("Error creating backup: %v", err), "", nil
	}

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if exists {
		cyan.Printf("⬆️  Prepend to File / ファイル先頭に追記\n")
	} else {
		cyan.Printf("⬆️  Create File with Content / ファイルを作成\n")
	}
	cyan.Printf("📂 Path / パス: %s\n", path)
	contentLines := strings.Split(content, "\n")
	cyan.Printf("📏 Adding / 追加: %d lines / 行\n", len(contentLines))
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	yellow.Println("\nContent to prepend / 先頭に追加する内容:")
	cyan.Println("┌" + strings.Repeat("─", 60) + "┐")
	for i, line := range contentLines {
		if i >= 10 {
			yellow.Printf("│ ... (%d more lines / 行省略)\n", len(contentLines)-10)
			break
		}
		green.Printf("│ + %s\n", line)
	}
	cyan.Println("└" + strings.Repeat("─", 60) + "┘")

	if exists && len(oldContent) > 0 {
		oldLines := strings.Split(string(oldContent), "\n")
		yellow.Println("\nExisting file (first 10 lines) / 既存ファイル（最初10行）:")
		cyan.Println("┌" + strings.Repeat("─", 60) + "┐")
		for i := 0; i < len(oldLines) && i < 10; i++ {
			fmt.Printf("│ %s\n", oldLines[i])
		}
		cyan.Println("└" + strings.Repeat("─", 60) + "┘")
	}
	fmt.Println()

	newContent := content
	if !strings.HasSuffix(content, "\n") {
		newContent += "\n"
	}
	newContent += string(oldContent)

	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("Error writing file: %v", err), "", nil
	}

	green.Printf("✅ Prepended: %s\n", path)
	return fmt.Sprintf("Successfully prepended %d bytes to %s", len(content), path), backupPath, nil
}
