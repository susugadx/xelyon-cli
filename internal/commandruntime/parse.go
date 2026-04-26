package commandruntime

import "strings"

// Invocation は parse 済み slash command 呼び出しを表す。
type Invocation struct {
	Command string
	Args    []string
}

// Split は slash command 文字列を quote-aware に分割する。
func Split(input string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range input {
		switch {
		case (r == '"' || r == '\'') && !inQuote:
			inQuote = true
			quoteChar = r
		case r == quoteChar && inQuote:
			inQuote = false
			quoteChar = 0
		case r == ' ' && !inQuote:
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
	return parts
}

// Parse は input を分割し、先頭 command を alias 解決した Invocation にする。
func Parse(input string, userAliases map[string]string) (Invocation, bool) {
	parts := Split(input)
	if len(parts) == 0 {
		return Invocation{}, false
	}
	return Invocation{
		Command: ResolveAlias(parts[0], userAliases),
		Args:    append([]string(nil), parts[1:]...),
	}, true
}
