package gathercontext

import "strings"

type queryToken struct {
	text  string
	start int
}

func topLevelQueryTokens(query string) []queryToken {
	tokens := make([]queryToken, 0)
	start := -1
	var quote byte
	escaped := false

	for i := 0; i < len(query); i++ {
		c := query[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}

		switch {
		case c == '"' || c == '\'':
			if start < 0 {
				start = i
			}
			quote = c
		case isQueryWhitespace(c):
			if start >= 0 {
				tokens = append(tokens, queryToken{text: query[start:i], start: start})
				start = -1
			}
		default:
			if start < 0 {
				start = i
			}
		}
	}
	if start >= 0 {
		tokens = append(tokens, queryToken{text: query[start:], start: start})
	}
	return tokens
}

func splitQueryOnTopLevelWord(query, word string) ([]string, bool) {
	tokens := topLevelQueryTokens(query)
	if len(tokens) < 3 {
		return nil, false
	}

	parts := make([]string, 0, 2)
	start := 0
	found := false
	for _, token := range tokens {
		if !strings.EqualFold(token.text, word) {
			continue
		}
		parts = append(parts, query[start:token.start])
		start = token.start + len(token.text)
		found = true
	}
	if !found {
		return nil, false
	}
	parts = append(parts, query[start:])
	return parts, true
}

func unquoteQueryLiteral(value string) (string, bool) {
	if len(value) < 2 {
		return "", false
	}
	quote := value[0]
	if quote != '"' && quote != '\'' {
		return "", false
	}
	if value[len(value)-1] != quote || isEscapedQuote(value, len(value)-1) {
		return "", false
	}
	return value[1 : len(value)-1], true
}

func trimQueryBoundaryQuotes(value string) string {
	value = strings.TrimSpace(value)
	if unquoted, ok := unquoteQueryLiteral(value); ok {
		return strings.TrimSpace(unquoted)
	}
	return strings.TrimSpace(strings.Trim(value, `"'`))
}

func isEscapedQuote(value string, idx int) bool {
	backslashes := 0
	for i := idx - 1; i >= 0 && value[i] == '\\'; i-- {
		backslashes++
	}
	return backslashes%2 == 1
}

func isQueryWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}
