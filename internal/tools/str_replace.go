package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// executeStrReplace はファイル内の文字列を置換
func executeStrReplace(path string, oldStr string, newStr string) (string, string, error) {
	if path == "" {
		return "Error: path is required", "", nil
	}
	if oldStr == "" {
		return "Error: old_str is required", "", nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), "", nil
	}

	// ファイルを読み込む
	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err), "", nil
	}

	oldContent := string(content)
	var newContent string

	// まず完全一致を試行
	exactMatch := strings.Contains(oldContent, oldStr)
	exactCount := strings.Count(oldContent, oldStr)

	if exactMatch && exactCount == 1 {
		// 完全一致が1つ → そのまま使用
		newContent = strings.Replace(oldContent, oldStr, newStr, 1)
	} else if exactMatch && exactCount > 1 {
		// 完全一致が複数 → エラー
		lines := strings.Split(oldContent, "\n")
		previewLines := min(50, len(lines))
		preview := strings.Join(lines[:previewLines], "\n")

		return fmt.Sprintf(`Error: old_str appears %d times in %s (must be unique).

Hint: Include more context (surrounding lines) to make old_str unique.
For example, include the function signature or class definition.

File preview (first %d lines):
---
%s
---

Please use read_file to see the full content and choose a unique old_str.`,
			exactCount, path, previewLines, preview), "", nil
	} else {
		// 完全一致しない → 正規化マッチを試行
		yellow.Println("⚠️  Exact match failed, trying normalized whitespace matching...")

		found, startIdx, endIdx := findWithNormalizedWhitespace(oldContent, oldStr)

		if !found {
			return fmt.Sprintf("Error: old_str not found in %s (tried both exact and normalized matching)", path), "", nil
		}

		// 正規化マッチで見つかった部分を置換
		actualOldStr := oldContent[startIdx : endIdx+1]
		newContent = oldContent[:startIdx] + newStr + oldContent[endIdx+1:]

		yellow.Printf("ℹ️  Matched with normalized whitespace (indentation may differ)\n")
		yellow.Printf("   Actual match in file:\n")
		// 実際のマッチ部分をプレビュー表示
		matchLines := strings.Split(actualOldStr, "\n")
		for i, line := range matchLines {
			if i >= 5 {
				yellow.Printf("   ... (%d more lines)\n", len(matchLines)-5)
				break
			}
			yellow.Printf("   │ %s\n", line)
		}
		fmt.Println()
	}

	// 確認UI
	oldStrLines := strings.Split(oldStr, "\n")
	newStrLines := strings.Split(newStr, "\n")

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("🔧 Text Replacement / テキスト置換\n")
	cyan.Printf("📂 File / ファイル: %s\n", path)
	cyan.Printf("📊 Changes / 変更: -%d lines, +%d lines / %d 行削除・%d 行追加\n",
		len(oldStrLines), len(newStrLines), len(oldStrLines), len(newStrLines))
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 改善された差分表示
	showImprovedDiff(oldStr, newStr)

	// 確認
	if !confirm("Apply this replacement? / この置換を適用しますか？") {
		yellow.Println("⚠️  User cancelled the replacement")
		return fmt.Sprintf(`[CANCELLED] User cancelled str_replace for %s.

Hint: The replacement was not applied. If you need to make this change:
1. Check if the old_str is correct by using read_file
2. Try a smaller, more specific replacement
3. Ask the user for clarification

Do not retry the same replacement.`, path), "", nil
	}

	// バックアップ作成
	backupPath, err := createBackup(absPath)
	if err != nil {
		return fmt.Sprintf("Warning: failed to create backup: %v (continuing anyway)", err), "", nil
	}

	// 保存
	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("Error writing file: %v", err), "", nil
	}

	green.Printf("✅ Replaced in: %s\n", path)
	return fmt.Sprintf("Successfully replaced text in %s", path), backupPath, nil
}
