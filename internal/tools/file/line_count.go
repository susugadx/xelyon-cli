package file

import "strings"

// countLines は文字列の行数を概算する（\n区切り）。
// 空文字は 0 行として扱う。
func countLines(s string) int {
	if s == "" {
		return 0
	}
	// 末尾が改行でも1行として数える（splitは最後に空要素を作るので除外）
	parts := strings.Split(s, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		return len(parts) - 1
	}
	return len(parts)
}
