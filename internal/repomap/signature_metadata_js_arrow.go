package repomap

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func extractJSArrowFunctionMetadata(sig string) (string, string, bool) {
	rest := strings.TrimSpace(sig)
	if next, ok := consumeJSKeyword(rest, "export"); ok {
		var constOK bool
		rest, constOK = consumeJSKeyword(next, "const")
		if !constOK {
			return "", "", false
		}
	} else if next, ok := consumeJSKeyword(rest, "const"); ok {
		rest = next
	} else if next, ok := consumeJSKeyword(rest, "let"); ok {
		rest = next
	} else {
		return "", "", false
	}

	nameEnd := 0
	for i, r := range rest {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9' && i > 0) || r == '_' {
			nameEnd = i + 1
			continue
		}
		break
	}
	if nameEnd == 0 {
		return "", "", false
	}

	name := rest[:nameEnd]
	rest = strings.TrimSpace(rest[nameEnd:])
	if strings.HasPrefix(rest, ":") {
		eqIdx := indexJSAssignmentAfterTypeAnnotation(rest)
		if eqIdx < 0 {
			return "", "", false
		}
		rest = strings.TrimSpace(rest[eqIdx:])
	}
	if !strings.HasPrefix(rest, "=") {
		return "", "", false
	}

	rest = strings.TrimSpace(strings.TrimPrefix(rest, "="))
	if next, ok := consumeJSAsyncArrowModifier(rest); ok {
		rest = next
	}
	if next, ok := consumeJSTypeParameters(rest); ok {
		rest = next
	}

	if strings.HasPrefix(rest, "(") {
		closeIdx := strings.Index(rest, ")")
		if closeIdx < 0 {
			return "", "", false
		}
		if !strings.Contains(rest[closeIdx+1:], "=>") {
			return "", "", false
		}
		return name, "function", true
	}

	if !isJSSingleParamArrow(rest) {
		return "", "", false
	}

	return name, "function", true
}

func consumeJSTypeParameters(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "<") {
		return "", false
	}

	depth := 0
	for i, r := range s {
		switch r {
		case '<':
			depth++
		case '>':
			if i > 0 && s[i-1] == '=' {
				continue
			}
			if depth > 0 {
				depth--
			}
			if depth == 0 {
				return strings.TrimSpace(s[i+1:]), true
			}
		}
	}
	return "", false
}

func indexJSAssignmentAfterTypeAnnotation(s string) int {
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	angleDepth := 0

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case '<':
			angleDepth++
		case '>':
			if i > 0 && s[i-1] == '=' {
				continue
			}
			if angleDepth > 0 {
				angleDepth--
			}
		case '=':
			if i+1 < len(s) && s[i+1] == '>' {
				continue
			}
			if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && angleDepth == 0 {
				return i
			}
		}
	}
	return -1
}

func isJSSingleParamArrow(s string) bool {
	_, rest, ok := consumeJSIdentifierToken(strings.TrimSpace(s))
	if !ok {
		return false
	}
	rest = strings.TrimLeftFunc(rest, unicode.IsSpace)
	return strings.HasPrefix(rest, "=>")
}

func consumeJSAsyncArrowModifier(s string) (string, bool) {
	next, ok := consumeJSKeyword(s, "async")
	if !ok || strings.HasPrefix(next, "=>") {
		return "", false
	}
	return next, true
}

func consumeJSIdentifierToken(s string) (string, string, bool) {
	if s == "" {
		return "", "", false
	}

	r, size := utf8.DecodeRuneInString(s)
	if !isJSIdentifierStart(r) {
		return "", "", false
	}

	end := size
	for end < len(s) {
		r, size = utf8.DecodeRuneInString(s[end:])
		if !isJSIdentifierPart(r) {
			break
		}
		end += size
	}
	return s[:end], s[end:], true
}

func isJSIdentifierStart(r rune) bool {
	return r == '_' || r == '$' || unicode.IsLetter(r)
}

func isJSIdentifierPart(r rune) bool {
	return isJSIdentifierStart(r) || unicode.IsDigit(r)
}

func consumeJSKeyword(s, keyword string) (string, bool) {
	if !strings.HasPrefix(s, keyword) {
		return "", false
	}
	tail := s[len(keyword):]
	if tail == "" {
		return "", true
	}
	r, size := utf8.DecodeRuneInString(tail)
	if !unicode.IsSpace(r) {
		return "", false
	}
	return strings.TrimLeftFunc(tail[size:], unicode.IsSpace), true
}
