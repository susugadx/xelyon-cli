package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// executeInsertBefore はパターンマッチした行の前に内容を挿入
func executeInsertBefore(path, pattern, content string) (string, string, error) {
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

	// Tier 1: Exact match
	matchIdx := -1
	matchCount := 0
	var matchIndices []int

	for i, line := range lines {
		if line == pattern {
			matchIdx = i
			matchCount++
			matchIndices = append(matchIndices, i)
		}
	}

	// Tier 2: Normalized whitespace match
	if matchIdx == -1 {
		normalizedPattern := normalizeLeadingWhitespace(pattern)
		for i, line := range lines {
			if normalizeLeadingWhitespace(line) == normalizedPattern {
				matchIdx = i
				matchCount++
				matchIndices = append(matchIndices, i)
			}
		}
	}

	// パターンが見つからない場合
	if matchIdx == -1 {
		yellow.Printf("⚠️  Pattern not found / パターンが見つかりません: %s\n\n", pattern)
		yellow.Println("File preview (first 50 lines) / ファイルプレビュー (最初50行):")
		for i := 0; i < min(len(lines), 50); i++ {
			fmt.Printf("%4d: %s\n", i+1, lines[i])
		}
		if len(lines) > 50 {
			yellow.Printf("... (%d more lines)\n", len(lines)-50)
		}
		return fmt.Sprintf("Error: Pattern not found in %s", path), "", nil
	}

	// 複数マッチの場合
	if matchCount > 1 {
		red.Printf("⚠️  Error: Pattern matches %d locations (must be unique)\n", matchCount)
		red.Println("⚠️  エラー: パターンが複数の場所にマッチします（一意である必要があります）")
		yellow.Println("All match locations / すべてのマッチ場所:")
		for _, idx := range matchIndices {
			start := max(0, idx-2)
			end := min(len(lines), idx+3)
			for i := start; i < end; i++ {
				prefix := "  "
				if i == idx {
					prefix = "→ "
				}
				fmt.Printf("%s%4d: %s\n", prefix, i+1, lines[i])
			}
			fmt.Println()
		}
		return fmt.Sprintf("Error: Pattern matched %d times (must be unique)", matchCount), "", nil
	}

	// コンテキスト表示（マッチ行の前後5行）
	green.Printf("✅ Pattern found at line %d / パターンが見つかりました (行 %d)\n\n", matchIdx+1, matchIdx+1)
	yellow.Println("Context / コンテキスト (5 lines before/after):")
	start := max(0, matchIdx-5)
	end := min(len(lines), matchIdx+6)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == matchIdx {
			prefix = "→ "
		}
		fmt.Printf("%s%4d: %s\n", prefix, i+1, lines[i])
	}
	cyan.Println("\n━━━━ Content to insert / 挿入する内容 ━━━━")
	fmt.Println(content)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// バックアップ作成
	backupPath, err := createBackup(absPath)
	if err != nil {
		return fmt.Sprintf("Error: Failed to create backup: %v", err), "", nil
	}
	green.Printf("📦 Backup created: %s\n", backupPath)

	// 挿入実行
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:matchIdx]...)
	newLines = append(newLines, content)
	newLines = append(newLines, lines[matchIdx:]...)

	newContent := strings.Join(newLines, "\n")
	err = os.WriteFile(absPath, []byte(newContent), 0644)
	if err != nil {
		return fmt.Sprintf("Error: Failed to write file: %v", err), "", nil
	}

	return fmt.Sprintf("✅ Inserted before line %d in %s", matchIdx+1, path), backupPath, nil
}
