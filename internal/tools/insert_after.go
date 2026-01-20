package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// executeInsertAfter はパターンマッチした行の後に内容を挿入
func executeInsertAfter(path, pattern, content string) (string, string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: Invalid file path: %v", err), "", nil
	}

	// ファイル読み込み
	fileContent, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Sprintf("Error: Cannot read file: %v", err), "", nil
	}

	lines := strings.Split(string(fileContent), "\n")

	// パターンマッチング（共通関数使用）
	result := FindPatternInLines(lines, pattern)

	// パターンが見つからない場合
	if result.MatchIdx == -1 {
		DisplayPatternNotFound(pattern, lines, 50)
		return fmt.Sprintf("Error: Pattern not found in %s", path), "", nil
	}

	// 複数マッチの場合
	if result.MatchCount > 1 {
		DisplayMultipleMatches(result.MatchIndices, lines)
		return fmt.Sprintf("Error: Pattern matched %d times (must be unique)", result.MatchCount), "", nil
	}

	// コンテキスト表示
	DisplayContextAround(result.MatchIdx, lines, 5)
	DisplayContentToInsert(content)

	// バックアップ作成
	backupPath, err := createBackup(absPath)
	if err != nil {
		return fmt.Sprintf("Error: Failed to create backup: %v", err), "", nil
	}
	green.Printf("📦 Backup created: %s\n", backupPath)

	// 挿入実行（マッチ行の後に挿入）
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:result.MatchIdx+1]...)
	newLines = append(newLines, content)
	newLines = append(newLines, lines[result.MatchIdx+1:]...)

	newContent := strings.Join(newLines, "\n")
	err = os.WriteFile(absPath, []byte(newContent), 0644)
	if err != nil {
		return fmt.Sprintf("Error: Failed to write file: %v", err), "", nil
	}

	return fmt.Sprintf("✅ Inserted after line %d in %s", result.MatchIdx+1, path), backupPath, nil
}
