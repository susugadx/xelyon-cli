package jsast

import "strings"

type fallbackDefaultWrappedSymbol struct {
	name       string
	kind       string
	nameColumn int
}

func fallbackDefaultWrappedSymbolFromLine(parsed *ParsedFile, line string, trimmed string, lineNo int, lineStart int) (Symbol, bool) {
	scanned, ok := scanFallbackDefaultWrappedSymbol(trimmed)
	if !ok {
		return Symbol{}, false
	}
	trimmedColumn := strings.Index(line, trimmed)
	if trimmedColumn < 0 {
		trimmedColumn = 0
	}
	return fallbackSymbolAtNameColumn(parsed, trimmed, lineNo, lineStart, trimmedColumn+scanned.nameColumn, scanned.name, scanned.kind, true), true
}

func scanFallbackDefaultWrappedSymbol(trimmed string) (fallbackDefaultWrappedSymbol, bool) {
	scanner := fallbackDefaultWrapperScanner{text: trimmed}
	if !scanner.consumeKeyword("export") {
		return fallbackDefaultWrappedSymbol{}, false
	}
	scanner.skipSpaces()
	if !scanner.consumeKeyword("default") {
		return fallbackDefaultWrappedSymbol{}, false
	}
	scanner.skipSpaces()
	if !scanner.consumeWrapperCalls() {
		return fallbackDefaultWrappedSymbol{}, false
	}
	scanner.skipSpaces()

	var kind string
	if scanner.consumeKeyword("async") {
		scanner.skipSpaces()
	}
	switch {
	case scanner.consumeKeyword("function"):
		kind = "function"
	case scanner.consumeKeyword("class"):
		kind = "class"
	default:
		return fallbackDefaultWrappedSymbol{}, false
	}
	scanner.skipSpaces()
	nameStart := scanner.pos
	name, ok := scanner.consumeIdentifier()
	if !ok || name == "" {
		return fallbackDefaultWrappedSymbol{}, false
	}
	return fallbackDefaultWrappedSymbol{name: name, kind: kind, nameColumn: nameStart}, true
}

type fallbackDefaultWrapperScanner struct {
	text string
	pos  int
}

func (s *fallbackDefaultWrapperScanner) consumeWrapperCalls() bool {
	count := 0
	for {
		start := s.pos
		if !s.consumeWrapperCallee() {
			s.pos = start
			break
		}
		s.skipSpaces()
		if s.peekByte() == '<' {
			if !s.consumeTypeArguments() {
				return false
			}
			s.skipSpaces()
		}
		if !s.consumeByte('(') {
			return false
		}
		count++
		s.skipSpaces()
	}
	return count > 0
}

func (s *fallbackDefaultWrapperScanner) consumeWrapperCallee() bool {
	start := s.pos
	name, ok := s.consumeIdentifier()
	if !ok {
		return false
	}
	if name == "React" {
		s.skipSpaces()
		if !s.consumeByte('.') {
			s.pos = start
			return false
		}
		s.skipSpaces()
		name, ok = s.consumeIdentifier()
		if !ok {
			s.pos = start
			return false
		}
	}
	switch name {
	case "memo", "forwardRef":
		return true
	default:
		s.pos = start
		return false
	}
}

func (s *fallbackDefaultWrapperScanner) consumeTypeArguments() bool {
	if !s.consumeByte('<') {
		return false
	}
	depth := 1
	var quote byte
	escaped := false
	for s.pos < len(s.text) {
		ch := s.text[s.pos]
		s.pos++
		if quote != 0 {
			switch {
			case escaped:
				escaped = false
			case ch == '\\':
				escaped = true
			case ch == quote:
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"', '`':
			quote = ch
		case '<':
			depth++
		case '>':
			if s.pos >= 2 && s.text[s.pos-2] == '=' {
				continue
			}
			depth--
			if depth == 0 {
				return true
			}
		}
	}
	return false
}

func (s *fallbackDefaultWrapperScanner) consumeKeyword(keyword string) bool {
	start := s.pos
	got, ok := s.consumeIdentifier()
	if !ok || got != keyword {
		s.pos = start
		return false
	}
	return true
}

func (s *fallbackDefaultWrapperScanner) consumeIdentifier() (string, bool) {
	if s.pos >= len(s.text) || !fallbackIdentifierStartByte(s.text[s.pos]) {
		return "", false
	}
	start := s.pos
	s.pos++
	for s.pos < len(s.text) && fallbackIdentifierByte(s.text[s.pos]) {
		s.pos++
	}
	return s.text[start:s.pos], true
}

func (s *fallbackDefaultWrapperScanner) skipSpaces() {
	for s.pos < len(s.text) {
		switch s.text[s.pos] {
		case ' ', '\t', '\r', '\n':
			s.pos++
		default:
			return
		}
	}
}

func (s *fallbackDefaultWrapperScanner) consumeByte(want byte) bool {
	if s.peekByte() != want {
		return false
	}
	s.pos++
	return true
}

func (s *fallbackDefaultWrapperScanner) peekByte() byte {
	if s.pos >= len(s.text) {
		return 0
	}
	return s.text[s.pos]
}

func fallbackIdentifierStartByte(ch byte) bool {
	return ch == '_' || ch == '$' || ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z')
}

func fallbackIdentifierByte(ch byte) bool {
	return fallbackIdentifierStartByte(ch) || ('0' <= ch && ch <= '9')
}
