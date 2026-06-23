package rawoutputs

import (
	"regexp"
	"strings"
)

const (
	tokenKeyPrefixPattern       = `(^|[^A-Za-z0-9_.-])`
	doubleQuotedTokenKeyPattern = `"([^"\r\n]*token\s*)"`
	singleQuotedTokenKeyPattern = `'([^'\r\n]*token\s*)'`
	unquotedTokenKeyPattern     = `([A-Za-z0-9_.-]*token)`
	tokenKeySeparatorPattern    = `\s*([:=])\s*`

	tokenKeyPatternIndexCount    = 12
	tokenKeyPatternPrefixStart   = 2
	tokenKeyPatternPrefixEnd     = 3
	tokenKeyPatternSeparatorFrom = 10
	tokenKeyPatternSeparatorTo   = 11
)

var (
	tokenKeyPattern = regexp.MustCompile(`(?i)` +
		tokenKeyPrefixPattern +
		`(?:` + doubleQuotedTokenKeyPattern + `|` + singleQuotedTokenKeyPattern + `|` + unquotedTokenKeyPattern + `)` +
		tokenKeySeparatorPattern)
	tokenKeyPatternKeySpans = [][2]int{{4, 5}, {6, 7}, {8, 9}}
)

func tokenKeyValueLooksSensitive(value string) bool {
	_, ok := firstTokenKeyValue(value)
	return ok
}

func redactTokenKeyValues(value string) string {
	var b strings.Builder
	last := 0
	searchStart := 0
	for searchStart < len(value) {
		match, ok := firstTokenKeyValue(value[searchStart:])
		if !ok {
			break
		}
		match = match.offset(searchStart)
		if match.valueEnd < match.fullEnd {
			break
		}
		b.WriteString(value[last:match.fullStart])
		b.WriteString(value[match.prefixStart:match.prefixEnd])
		b.WriteString(match.key)
		if match.separator == "=" {
			b.WriteString("=[redacted]")
		} else {
			b.WriteString(": [redacted]")
		}
		last = match.valueEnd
		searchStart = match.valueEnd
	}
	if last == 0 {
		return value
	}
	b.WriteString(value[last:])
	return b.String()
}

type tokenKeyValueMatch struct {
	fullStart   int
	fullEnd     int
	prefixStart int
	prefixEnd   int
	key         string
	separator   string
	valueEnd    int
}

func (m tokenKeyValueMatch) offset(delta int) tokenKeyValueMatch {
	m.fullStart += delta
	m.fullEnd += delta
	m.prefixStart += delta
	m.prefixEnd += delta
	m.valueEnd += delta
	return m
}

func firstTokenKeyValue(value string) (tokenKeyValueMatch, bool) {
	match := tokenKeyPattern.FindStringSubmatchIndex(value)
	if len(match) < tokenKeyPatternIndexCount {
		return tokenKeyValueMatch{}, false
	}
	keyStart, keyEnd := tokenKeyMatchKeySpan(match)
	if keyStart < 0 || keyEnd < 0 {
		return tokenKeyValueMatch{}, false
	}
	key := strings.TrimSpace(value[keyStart:keyEnd])
	if !sensitiveTokenKey(key) {
		return tokenKeyValueMatch{}, false
	}
	return tokenKeyValueMatch{
		fullStart:   match[0],
		fullEnd:     match[1],
		prefixStart: match[tokenKeyPatternPrefixStart],
		prefixEnd:   match[tokenKeyPatternPrefixEnd],
		key:         key,
		separator:   value[match[tokenKeyPatternSeparatorFrom]:match[tokenKeyPatternSeparatorTo]],
		valueEnd:    tokenValueEnd(value, match[1]),
	}, true
}

func tokenKeyMatchKeySpan(match []int) (int, int) {
	for _, span := range tokenKeyPatternKeySpans {
		start, end := match[span[0]], match[span[1]]
		if start >= 0 && end >= 0 {
			return start, end
		}
	}
	return -1, -1
}

func tokenValueEnd(value string, start int) int {
	if start >= len(value) {
		return start
	}
	switch value[start] {
	case '"', '\'':
		return quotedTokenValueEnd(value, start, value[start])
	case '[', '{':
		return structuredTokenValueEnd(value, start)
	default:
		return scalarTokenValueEnd(value, start)
	}
}

func quotedTokenValueEnd(value string, start int, quote byte) int {
	escaped := false
	for i := start + 1; i < len(value); i++ {
		switch {
		case escaped:
			escaped = false
		case value[i] == '\\':
			escaped = true
		case value[i] == quote:
			return i + 1
		case value[i] == '\r' || value[i] == '\n':
			return i
		}
	}
	return len(value)
}

func structuredTokenValueEnd(value string, start int) int {
	open := value[start]
	close := byte('}')
	if open == '[' {
		close = ']'
	}
	depth := 0
	var quote byte
	escaped := false
	for i := start; i < len(value); i++ {
		if quote != 0 {
			switch {
			case escaped:
				escaped = false
			case value[i] == '\\':
				escaped = true
			case value[i] == quote:
				quote = 0
			}
			continue
		}
		switch value[i] {
		case '"', '\'':
			quote = value[i]
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return len(value)
}

func scalarTokenValueEnd(value string, start int) int {
	if hasBearerPrefix(value, start) {
		i := start + len("bearer")
		for i < len(value) && (value[i] == ' ' || value[i] == '\t') {
			i++
		}
		for i < len(value) && !tokenScalarDelimiter(value[i]) {
			i++
		}
		return i
	}
	i := start
	for i < len(value) && !tokenScalarDelimiter(value[i]) {
		i++
	}
	return i
}

func hasBearerPrefix(value string, start int) bool {
	if len(value[start:]) < len("bearer") {
		return false
	}
	if !strings.EqualFold(value[start:start+len("bearer")], "bearer") {
		return false
	}
	next := start + len("bearer")
	return next < len(value) && (value[next] == ' ' || value[next] == '\t')
}

func tokenScalarDelimiter(value byte) bool {
	switch value {
	case ' ', '\t', '\r', '\n', '&', ';', '[', ']', '}', ')', ',':
		return true
	default:
		return false
	}
}

func sensitiveTokenKey(key string) bool {
	key = strings.Trim(strings.TrimSpace(key), `"'`)
	if key == "" {
		return false
	}
	return strings.HasSuffix(strings.ToLower(key), "token")
}
