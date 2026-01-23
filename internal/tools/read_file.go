package tools

import (
	"fmt"
	"os"
	"strings"
)

// MaxReadLines はデフォルトの最大読み込み行数
const MaxReadLines = 200

// executeReadFile はファイルを読み込む（行範囲指定対応）
// startLine, endLine が指定されている場合はその範囲のみ返す
// 指定がない場合は最初のMaxReadLines行を返す
func executeReadFile(path string, startLine, endLine int) string {
	if path == "" {
		return "Error: path is empty"
	}

	// パストラバーサル防止
	absPath, err := ValidatePath(path)
	if err != nil {
		red.Printf("🚫 Security: %v\n", err)
		return fmt.Sprintf("Error: %v", err)
	}

	var contentStr string

	// キャッシュチェック（行範囲指定なしの場合のみ）
	if startLine == 0 && endLine == 0 && GlobalToolCache != nil {
		if cached, hit := GlobalToolCache.GetFile(absPath); hit {
			contentStr = cached
		}
	}

	// キャッシュミスまたは行範囲指定ありの場合はファイルを読む
	if contentStr == "" {
		content, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Sprintf("Error reading file: %v", err)
		}
		contentStr = string(content)

		// キャッシュに保存（行範囲指定なしの場合のみ）
		if startLine == 0 && endLine == 0 && GlobalToolCache != nil {
			GlobalToolCache.SetFile(absPath, contentStr)
		}
	}

	lines := strings.Split(contentStr, "\n")
	totalLines := len(lines)

	// 行範囲が指定されている場合
	if startLine > 0 && endLine > 0 {
		// 範囲調整
		if startLine > totalLines {
			return fmt.Sprintf("Error: start_line %d exceeds total lines %d", startLine, totalLines)
		}
		if endLine > totalLines {
			endLine = totalLines
		}
		if startLine > endLine {
			return fmt.Sprintf("Error: start_line %d is greater than end_line %d", startLine, endLine)
		}

		// 1-indexed to 0-indexed
		selectedLines := lines[startLine-1 : endLine]
		result := formatLinesWithNumbers(selectedLines, startLine)
		green.Printf("📄 Read: %s (lines %d-%d of %d)\n", path, startLine, endLine, totalLines)
		return result
	}

	// 行範囲指定なし: デフォルトで最初のMaxReadLines行
	if totalLines <= MaxReadLines {
		// 全行表示
		result := formatLinesWithNumbers(lines, 1)
		green.Printf("📄 Read: %s (%d lines)\n", path, totalLines)
		return result
	}

	// 切り詰めて表示
	selectedLines := lines[:MaxReadLines]
	result := formatLinesWithNumbers(selectedLines, 1)
	remaining := totalLines - MaxReadLines
	result += fmt.Sprintf("\n... (truncated, %d lines remaining)\nUse start_line/end_line to read specific sections.", remaining)
	green.Printf("📄 Read: %s (showing first %d of %d lines)\n", path, MaxReadLines, totalLines)
	return result
}

// formatLinesWithNumbers は行番号付きでフォーマット
func formatLinesWithNumbers(lines []string, startNum int) string {
	var sb strings.Builder
	for i, line := range lines {
		sb.WriteString(fmt.Sprintf("%d: %s\n", startNum+i, line))
	}
	return sb.String()
}
