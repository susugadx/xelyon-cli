package token

import "strings"

// TruncateToLastLines はテキストを最後のN行に切り詰める
// 切り詰めた場合は (切り詰め後テキスト, true) を返す
func TruncateToLastLines(text string, lastN int) (string, bool) {
	if lastN <= 0 {
		return text, false
	}
	lines := strings.Split(text, "\n")
	if len(lines) <= lastN {
		return text, false
	}
	start := len(lines) - lastN
	truncated := strings.Join(lines[start:], "\n")
	return truncated, true
}

// IsTokenLimitError はエラーがトークン上限エラーかどうかを判定
func IsTokenLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// OpenAI
	if strings.Contains(msg, "input tokens exceed") {
		return true
	}
	// Generic patterns (best-effort)
	if strings.Contains(msg, "context length") || strings.Contains(msg, "maximum context") {
		return true
	}
	return false
}
