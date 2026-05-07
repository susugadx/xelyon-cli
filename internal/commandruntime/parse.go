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
		case isQuoteRune(r) && !inQuote:
			// token 先頭、または語中でも対応する閉じ quote がある場合は quote セグメントとして扱う。
			// これにより foo'bar baz' のような shell-style 結合を維持しつつ、
			// don't のように閉じ quote がない語中 apostrophe は文字として保持できる。
			if shouldStartQuoteSegment(current.Len(), runes, i, r) {
				inQuote = true
				quoteChar = r
				continue
			}
			current.WriteRune(r)
		case r == '\\' && inQuote:
			i = consumeQuotedEscapeRune(runes, i, quoteChar, &current)
		case r == quoteChar && inQuote:
			inQuote = false
			quoteChar = 0
		case shouldSplitOnWhitespace(r, inQuote):
			flushCurrentToken(&parts, &current)
		default:
			current.WriteRune(r)
		}
	}
	flushCurrentToken(&parts, &current)
	if inQuote {
		return parts, SplitStatusUnterminatedQuote
	}
	return parts, SplitStatusOK
}

func isQuoteRune(r rune) bool {
	return r == '"' || r == '\''
}

func shouldSplitOnWhitespace(r rune, inQuote bool) bool {
	return unicode.IsSpace(r) && !inQuote
}

func flushCurrentToken(parts *[]string, current *strings.Builder) {
	if current.Len() == 0 {
		return
	}
	*parts = append(*parts, current.String())
	current.Reset()
}

func consumeQuotedEscapeRune(runes []rune, index int, quoteChar rune, current *strings.Builder) int {
	// quoted string では quote だけをエスケープ対象として扱う。
	// `\\` を縮退させると UNC path 先頭が壊れるため、`\` 自体は保持する。
	if index+1 >= len(runes) {
		current.WriteRune('\\')
		return index
	}
	next := runes[index+1]
	if next == quoteChar {
		current.WriteRune(next)
		return index + 1
	}
	current.WriteRune('\\')
	return index
}

// Parse は input を分割し、先頭 command と引数を返す。
// alias 解決は command catalog 側を source of truth とする。
func Parse(input string) (Invocation, bool) {
	parts, status := SplitStrict(input)
	if !status.IsOK() || len(parts) == 0 {
		return Invocation{}, false
	}
	return Invocation{
		Command: strings.ToLower(parts[0]),
		Args:    append([]string(nil), parts[1:]...),
	}, true
}
