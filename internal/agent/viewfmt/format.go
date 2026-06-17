package viewfmt

import (
	"fmt"
	"strings"
)

// Number は整数をカンマ区切りで表示する。
func Number(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%s,%03d", Number(n/1000), n%1000)
}

// Tokens は token count を status 表示用の短い K/M 形式に整形する。
func Tokens(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

// FileSize は byte count を人間が読める binary unit 表記に整形する。
func FileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// USD は USD 金額を小数 4 桁で表示する。
func USD(value float64) string {
	return fmt.Sprintf("$%.4f", value)
}

// USDWithSuffix は USD 金額に通貨 suffix を付けて表示する。
func USDWithSuffix(value float64) string {
	return fmt.Sprintf("%s USD", USD(value))
}

// FirstLine は複数行文字列の先頭行だけを返す。
func FirstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

// Truncate は文字列が最大長を超える場合に末尾を省略する。
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
