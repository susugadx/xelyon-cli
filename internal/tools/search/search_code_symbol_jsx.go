package search

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func isJSXElementUsageRef(ref genericSymbolRef, symbol string) bool {
	if !isJSXSourceFile(ref.File) {
		return false
	}
	return isJSXElementUsageSnippet(ref.Snippet, symbol)
}

func isJSXSourceFile(path string) bool {
	path = strings.ToLower(strings.TrimSpace(path))
	return strings.HasSuffix(path, ".tsx") || strings.HasSuffix(path, ".jsx")
}

func isJSXElementUsageSnippet(snippet, symbol string) bool {
	symbol = strings.TrimSpace(symbol)
	if !isJSXComponentSymbol(symbol) {
		return false
	}

	for i := 0; i < len(snippet); i++ {
		if snippet[i] != '<' {
			continue
		}
		if isJSXOpeningElementAt(snippet, i, symbol) {
			return true
		}
	}
	return false
}

func isJSXOpeningElementAt(snippet string, index int, symbol string) bool {
	if isIgnoredJSXSnippetLocation(snippet, index) || isLikelyJSTypeArgumentPrefix(snippet, index) {
		return false
	}

	nameStart := skipJSXWhitespace(snippet, index+1)
	if nameStart >= len(snippet) || snippet[nameStart] == '/' {
		return false
	}
	if !strings.HasPrefix(snippet[nameStart:], symbol) {
		return false
	}

	nameEnd := nameStart + len(symbol)
	if nameEnd == len(snippet) {
		return true
	}
	return isJSXOpeningTagNameBoundary(snippet[nameEnd])
}

func skipJSXWhitespace(snippet string, index int) int {
	for index < len(snippet) && isJSXWhitespace(snippet[index]) {
		index++
	}
	return index
}

func isJSXOpeningTagNameBoundary(ch byte) bool {
	return isJSXWhitespace(ch) || ch == '/' || ch == '>' || ch == '<'
}

func isJSXWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f'
}

func isJSXComponentSymbol(symbol string) bool {
	if symbol == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(symbol)
	return unicode.IsUpper(r)
}

func isIgnoredJSXSnippetLocation(snippet string, index int) bool {
	inString, inLineComment, inBlockComment := jsSnippetContextAt(snippet, index)
	return inString || inLineComment || inBlockComment
}

func isLikelyJSTypeArgumentPrefix(snippet string, index int) bool {
	i := index - 1
	for i >= 0 && (snippet[i] == ' ' || snippet[i] == '\t') {
		i--
	}
	if i < 0 {
		return false
	}
	if snippet[i] == '.' || snippet[i] == ']' || snippet[i] == ')' {
		return true
	}
	if !isJSIdentifierPartByte(snippet[i]) {
		return false
	}
	token := jsIdentifierTokenEndingAt(snippet, i)
	switch token {
	case "return", "yield", "case":
		return false
	default:
		return true
	}
}

func jsIdentifierTokenEndingAt(snippet string, index int) string {
	start := index
	for start >= 0 && isJSIdentifierPartByte(snippet[start]) {
		start--
	}
	return snippet[start+1 : index+1]
}

func isJSIdentifierPartByte(ch byte) bool {
	return ch == '_' || ch == '$' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}

func jsSnippetContextAt(snippet string, index int) (inString, inLineComment, inBlockComment bool) {
	var quote byte
	escaped := false
	for i := 0; i < len(snippet) && i < index; i++ {
		ch := snippet[i]
		next := byte(0)
		if i+1 < len(snippet) {
			next = snippet[i+1]
		}

		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}

		if inBlockComment {
			if ch == '*' && next == '/' {
				inBlockComment = false
				i++
			}
			continue
		}

		if ch == '/' && next == '/' {
			return false, true, false
		}
		if ch == '/' && next == '*' {
			inBlockComment = true
			i++
			continue
		}
		if ch == '"' || ch == '\'' || ch == '`' {
			quote = ch
		}
	}
	return quote != 0, false, inBlockComment
}
