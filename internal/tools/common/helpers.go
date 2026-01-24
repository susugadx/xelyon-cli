package common

import (
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

// Colors - ui パッケージの共通色を使用
var (
	Yellow = ui.Yellow
	Green  = ui.Green
	Red    = ui.Red
	Cyan   = ui.Cyan
)

// GetCurrentTime returns the current time
func GetCurrentTime() time.Time {
	return time.Now()
}

// Truncate は文字列を指定長で切り詰め
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// NormalizeLeadingWhitespace は行頭の空白のみを正規化
// - タブをスペース4つに変換
// - 行頭の空白を削除
// - 行内の空白は保持（安全性重視）
func NormalizeLeadingWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	var normalized []string
	for _, line := range lines {
		// タブをスペース4つに変換
		line = strings.ReplaceAll(line, "\t", "    ")
		// 行頭の空白のみをトリム（行内は保持）
		trimmed := strings.TrimLeft(line, " ")
		normalized = append(normalized, trimmed)
	}
	return strings.Join(normalized, "\n")
}

// FindWithNormalizedWhitespace は正規化した状態で文字列を検索
func FindWithNormalizedWhitespace(content, pattern string) (found bool, startIdx, endIdx int) {
	normalizedContent := NormalizeLeadingWhitespace(content)
	normalizedPattern := NormalizeLeadingWhitespace(pattern)

	idx := strings.Index(normalizedContent, normalizedPattern)
	if idx == -1 {
		return false, -1, -1
	}

	// 正規化前の位置を計算（簡易実装：行番号ベース）
	contentLines := strings.Split(content, "\n")
	normalizedLines := strings.Split(normalizedContent, "\n")
	patternLines := strings.Split(normalizedPattern, "\n")

	// 正規化後の行番号を特定
	var currentPos int
	var lineNum int
	for i, line := range normalizedLines {
		if currentPos <= idx && idx < currentPos+len(line)+1 {
			lineNum = i
			break
		}
		currentPos += len(line) + 1 // +1 for \n
	}

	// 元のコンテンツから該当部分を抽出
	startLine := lineNum
	endLine := lineNum + len(patternLines) - 1

	if endLine >= len(contentLines) {
		return false, -1, -1
	}

	// 行単位で元の文字列を再構築
	var startPos int
	for i := 0; i < startLine; i++ {
		startPos += len(contentLines[i]) + 1
	}

	var endPos = startPos
	for i := startLine; i <= endLine; i++ {
		endPos += len(contentLines[i])
		if i < endLine {
			endPos += 1 // +1 for \n between lines
		}
	}

	return true, startPos, endPos - 1 // -1 because endIdx is inclusive
}

// Min は2つの整数の小さい方を返す
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max は2つの整数の大きい方を返す
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// FileExists はファイルが存在するかチェック
// NOTE: テスト用にグローバル変数としてオーバーライド可能
var FileExists = fileExistsImpl

func fileExistsImpl(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
