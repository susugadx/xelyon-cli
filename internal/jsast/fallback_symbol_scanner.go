package jsast

import "strings"

type fallbackSymbolScanner struct {
	parsed         *ParsedFile
	opts           fallbackSymbolOptions
	symbols        []Symbol
	byteOffset     int
	braceDepth     int
	typeBodyScopes []fallbackTypeBodyScope
	braceScanner   fallbackBraceScanner
}

func newFallbackSymbolScanner(parsed *ParsedFile, opts fallbackSymbolOptions) *fallbackSymbolScanner {
	return &fallbackSymbolScanner{
		parsed:         parsed,
		opts:           opts,
		symbols:        make([]Symbol, 0),
		typeBodyScopes: make([]fallbackTypeBodyScope, 0),
	}
}

func (s *fallbackSymbolScanner) Scan(lines []string) []Symbol {
	for idx, line := range lines {
		s.scanLine(idx+1, line)
	}
	return s.symbols
}

func (s *fallbackSymbolScanner) scanLine(lineNo int, line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed != "" && s.braceScanner.inCode() {
		s.addLineSymbols(line, trimmed, lineNo)
	}
	s.advanceLine(line)
}

func (s *fallbackSymbolScanner) addLineSymbols(line string, trimmed string, lineNo int) {
	if symbol, ok := fallbackSymbolFromLine(s.parsed, line, trimmed, lineNo, s.byteOffset); ok {
		s.symbols = append(s.symbols, symbol)
	}
	if scope, ok := fallbackDirectTypeBodyScope(s.braceDepth, s.typeBodyScopes); s.opts.includeTypeBodyMembers && ok {
		if symbol, ok := fallbackMethodSymbolFromLine(s.parsed, line, trimmed, lineNo, s.byteOffset); ok {
			s.symbols = append(s.symbols, symbol)
		}
		if symbol, ok := fallbackPropertySymbolFromLine(s.parsed, line, trimmed, lineNo, s.byteOffset, scope.kind); ok {
			s.symbols = append(s.symbols, symbol)
		}
	}
	if s.opts.includeTypeBodyMembers {
		if scope, ok := fallbackOpeningTypeBodyScope(trimmed, s.braceDepth+1); ok {
			s.typeBodyScopes = append(s.typeBodyScopes, scope)
		}
	}
}

func (s *fallbackSymbolScanner) advanceLine(line string) {
	s.braceDepth += s.braceScanner.delta(line)
	s.typeBodyScopes = fallbackOpenTypeBodyScopes(s.typeBodyScopes, s.braceDepth)
	s.byteOffset += len(line) + 1
}

type fallbackBraceScanner struct {
	blockComment bool
	regex        bool
	regexClass   bool
	quote        rune
	escaped      bool
}

func (s fallbackBraceScanner) inCode() bool {
	return !s.blockComment && !s.regex && s.quote == 0
}

func (s *fallbackBraceScanner) delta(line string) int {
	delta := 0
	for idx, r := range line {
		if s.blockComment {
			if r == '*' && nextRuneIs(line, idx, '/') {
				s.blockComment = false
			}
			continue
		}
		if s.quote != 0 {
			switch {
			case s.escaped:
				s.escaped = false
			case r == '\\':
				s.escaped = true
			case r == s.quote:
				s.quote = 0
			}
			continue
		}
		if s.regex {
			switch {
			case s.escaped:
				s.escaped = false
			case r == '\\':
				s.escaped = true
			case r == '[':
				s.regexClass = true
			case r == ']':
				s.regexClass = false
			case r == '/' && !s.regexClass:
				s.regex = false
			}
			continue
		}

		switch {
		case r == '/' && nextRuneIs(line, idx, '/'):
			return delta
		case r == '/' && nextRuneIs(line, idx, '*'):
			s.blockComment = true
		case r == '/' && fallbackSlashStartsRegex(line, idx):
			s.regex = true
		case r == '"', r == '\'', r == '`':
			s.quote = r
		case r == '{':
			delta++
		case r == '}':
			delta--
		}
	}
	if s.quote == '"' || s.quote == '\'' {
		s.quote = 0
		s.escaped = false
	}
	return delta
}

func fallbackSlashStartsRegex(line string, byteIdx int) bool {
	prev, ok := previousNonSpaceRune(line, byteIdx)
	if !ok {
		return true
	}
	switch prev {
	case '(', '[', '{', '=', ':', ',', ';', '!', '&', '|', '?', '+', '-', '*', '~', '^', '<', '>':
		return true
	}
	prefix := strings.TrimSpace(line[:byteIdx])
	for _, keyword := range []string{"return", "throw", "case", "delete", "typeof", "void", "yield"} {
		if strings.HasSuffix(prefix, keyword) {
			return true
		}
	}
	return false
}

func previousNonSpaceRune(line string, byteIdx int) (rune, bool) {
	var prev rune
	found := false
	for idx, r := range line {
		if idx >= byteIdx {
			break
		}
		if strings.TrimSpace(string(r)) == "" {
			continue
		}
		prev = r
		found = true
	}
	return prev, found
}

func nextRuneIs(line string, byteIdx int, want rune) bool {
	for idx, r := range line {
		if idx <= byteIdx {
			continue
		}
		return r == want
	}
	return false
}
