package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// executeDeleteLines deletes a range of lines from a file
func executeDeleteLines(path, startLineStr, endLineStr string) (string, string, error) {
	// 引数検証と変換
	startLine, err := strconv.Atoi(startLineStr)
	if err != nil {
		return fmt.Sprintf("Error: Invalid start_line: %v", err), "", nil
	}

	endLine, err := strconv.Atoi(endLineStr)
	if err != nil {
		return fmt.Sprintf("Error: Invalid end_line: %v", err), "", nil
	}

	// 範囲検証
	if startLine < 1 {
		return "Error: start_line must be >= 1", "", nil
	}
	if endLine < startLine {
		return "Error: end_line must be >= start_line", "", nil
	}

	// ファイル読み込み
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: Invalid path: %v", err), "", nil
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Sprintf("Error: Failed to read file: %v", err), "", nil
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return "Error: File is empty", "", nil
	}

	// endLineがファイル長を超える場合はクランプ（ユーザーフレンドリー）
	if endLine > len(lines) {
		endLine = len(lines)
	}

	// 削除される行数
	deleteCount := endLine - startLine + 1

	// 確認UI表示
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("🗑️  Delete Lines / 行削除\n")
	cyan.Printf("📂 File / ファイル: %s\n", path)
	cyan.Printf("📏 Lines / 行範囲: %d-%d (%d lines)\n", startLine, endLine, deleteCount)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	red.Println("⚠️  DESTRUCTIVE: Lines will be permanently deleted!")
	red.Println("⚠️  破壊的操作: 行が完全に削除されます!")

	// コンテキスト表示（削除行の前後5行）
	yellow.Println("\nContext / コンテキスト (5 lines before/after):")
	contextStart := startLine - 5
	if contextStart < 1 {
		contextStart = 1
	}
	contextEnd := endLine + 5
	if contextEnd > len(lines) {
		contextEnd = len(lines)
	}

	for i := contextStart; i <= contextEnd; i++ {
		if i >= startLine && i <= endLine {
			red.Printf("→ %4d: %s\n", i, lines[i-1]) // 削除される行は赤色
		} else {
			fmt.Printf("  %4d: %s\n", i, lines[i-1])
		}
	}

	dec := Confirm("Delete these lines? / これらの行を削除しますか？")
	switch dec.Action {
	case ConfirmYes:
		// continue
	case ConfirmComment:
		return fmt.Sprintf(`[COMMENT] User provided feedback for delete_lines.

Comment:
%s

Next actions:
- Re-check the line range (start_line/end_line).
- If the range is wrong, adjust the numbers and propose again.

IMPORTANT: Do NOT delete lines until the user approves.`, strings.TrimSpace(dec.Comment)), "", nil
	default:
		return "Cancelled by user", "", nil
	}

	// バックアップ作成
	backupPath, err := createBackup(absPath)
	if err != nil {
		return fmt.Sprintf("Error: Backup failed: %v", err), "", nil
	}
	green.Printf("📦 Backup created: %s\n", backupPath)

	// 新しい内容を構築（削除行を除外）
	newLines := make([]string, 0, len(lines)-deleteCount)
	newLines = append(newLines, lines[:startLine-1]...) // 削除前の行
	newLines = append(newLines, lines[endLine:]...)     // 削除後の行

	newContent := strings.Join(newLines, "\n")

	// ファイル書き込み
	err = os.WriteFile(absPath, []byte(newContent), 0644)
	if err != nil {
		return fmt.Sprintf("Error: Failed to write file: %v", err), "", nil
	}

	return fmt.Sprintf("✅ Deleted lines %d-%d from %s", startLine, endLine, path), backupPath, nil
}
