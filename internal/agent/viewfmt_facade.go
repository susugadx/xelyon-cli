package agent

import "github.com/susugadx/xelyon-cli/internal/agent/viewfmt"

// FormatFileSize はファイルサイズを人間が読める形式で返す。
func FormatFileSize(bytes int64) string {
	return viewfmt.FileSize(bytes)
}

// FormatTokens は K/M 形式でトークン数をフォーマットする。
func FormatTokens(n int) string {
	return viewfmt.Tokens(n)
}

// FormatNumber は数値にカンマを追加してフォーマットする。
func FormatNumber(n int) string {
	return viewfmt.Number(n)
}

func formatNumber(n int) string {
	return viewfmt.Number(n)
}
