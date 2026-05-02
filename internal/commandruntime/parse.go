package commandruntime

import (
	"strings"
	"unicode"
)

// Invocation は parse 済み slash command 呼び出しを表す。
type Invocation struct {
	Command string
	Args    []string
}

// SplitStatus は SplitStrict の parse 結果を表す。
type SplitStatus int

const (
	// SplitStatusOK は正常に parse できた状態を表す。
	SplitStatusOK SplitStatus = iota
	// SplitStatusUnterminatedQuote は未閉じ引用符を検出した状態を表す。
	SplitStatusUnterminatedQuote
)

// IsOK は status が正常かどうかを返す。
func (s SplitStatus) IsOK() bool {
	return s == SplitStatusOK
}

// ErrorSummary は parse 失敗時の簡潔な説明を返す。
func (s SplitStatus) ErrorSummary() string {
	switch s {
	case SplitStatusUnterminatedQuote:
		return "unmatched quote"
	default:
		return "invalid token"
	}
}

// Split は slash command 文字列を quote-aware に分割する。
func Split(input string) []string {
	parts, _ := SplitStrict(input)
	return parts
}

// SplitStrict は slash command 文字列を quote-aware に分割し、parse 状態を返す。
func SplitStrict(input string) ([]string, SplitStatus) {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	runes := []rune(input)
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		switch {
		case (r == '"' || r == '\'') && !inQuote:
			inQuote = true
			quoteChar = r
		case r == '\\' && inQuote:
			// quoted string では quote だけをエスケープ対象として扱う。
			// `\\` を縮退させると UNC path 先頭が壊れるため、`\` 自体は保持する。
			if i+1 < len(runes) {
				next := runes[i+1]
				if next == quoteChar {
					current.WriteRune(next)
					i++
					continue
				}
			}
			current.WriteRune('\\')
		case r == quoteChar && inQuote:
			inQuote = false
			quoteChar = 0
		case unicode.IsSpace(r) && !inQuote:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	if inQuote {
		return parts, SplitStatusUnterminatedQuote
	}
	return parts, SplitStatusOK
}

// Parse は input を分割し、先頭 command を alias 解決した Invocation にする。
func Parse(input string, userAliases map[string]string) (Invocation, bool) {
	parts, status := SplitStrict(input)
	if !status.IsOK() || len(parts) == 0 {
		return Invocation{}, false
	}
	return Invocation{
		Command: ResolveAlias(parts[0], userAliases),
		Args:    append([]string(nil), parts[1:]...),
	}, true
}
