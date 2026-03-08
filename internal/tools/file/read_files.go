package file

import (
	"fmt"
	"strconv"
	"strings"
)

// MaxReadFilesPaths は一度に読み込めるファイル数の上限
const MaxReadFilesPaths = 10

// parsePath はパス文字列を解析し、ファイルパスと行範囲を返す
// "path" → path, 0, 0
// "path:10" → path, 10, 0
// "path:10-20" → path, 10, 20
func parsePath(entry string) (string, int, int) {
	// 最後のコロンを探す（Windows パス C:\... 対策で lastIndex を使用）
	lastColon := strings.LastIndex(entry, ":")
	if lastColon < 0 {
		return entry, 0, 0
	}

	// コロン以降が数字または数字-数字でなければパスとして扱う
	suffix := entry[lastColon+1:]
	path := entry[:lastColon]

	// "start-end" 形式
	if dashIdx := strings.Index(suffix, "-"); dashIdx >= 0 {
		startStr := suffix[:dashIdx]
		endStr := suffix[dashIdx+1:]
		start, err1 := strconv.Atoi(startStr)
		end, err2 := strconv.Atoi(endStr)
		if err1 == nil && err2 == nil && start > 0 && end > 0 {
			return path, start, end
		}
		// パースできなければ全体をパスとして扱う
		return entry, 0, 0
	}

	// "start" のみ
	start, err := strconv.Atoi(suffix)
	if err == nil && start > 0 {
		return path, start, 0
	}

	// パースできなければ全体をパスとして扱う
	return entry, 0, 0
}

// perFileBudget はファイル数に応じた1ファイルあたりの最大行数を返す
func perFileBudget(n int) int {
	switch {
	case n <= 2:
		return MaxReadLines // 300
	case n <= 5:
		return 150
	default:
		return 80
	}
}

// ExecuteReadFiles は複数ファイルを一括読み込みする
func ExecuteReadFiles(paths []string) string {
	if len(paths) == 0 {
		return "Error: paths is empty"
	}
	if len(paths) > MaxReadFilesPaths {
		return fmt.Sprintf("Error: too many paths (max %d), got %d", MaxReadFilesPaths, len(paths))
	}

	budget := perFileBudget(len(paths))

	var sb strings.Builder
	for i, entry := range paths {
		if i > 0 {
			sb.WriteString("\n")
		}

		path, startLine, endLine := parsePath(entry)

		// 明示的な行範囲指定がない場合、バジェットを適用
		if startLine == 0 && endLine == 0 && budget < MaxReadLines {
			endLine = budget
		}

		// ファイルヘッダー
		fmt.Fprintf(&sb, "📄 File: %s\n", entry)

		// 既存の ExecuteReadFile を再利用
		result := ExecuteReadFile(path, startLine, endLine)
		sb.WriteString(result)
	}

	printReadStatus("📄 Read: %d files\n", len(paths))
	return sb.String()
}
