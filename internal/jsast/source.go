package jsast

import (
	"strings"
	"unicode/utf8"
)

type lexicalState int

const (
	lexNormal lexicalState = iota
	lexLineComment
	lexBlockComment
	lexSingleString
	lexDoubleString
	lexTemplateString
)

func lineByteRange(src []byte, line int) (uint, uint, bool) {
	if line <= 0 {
		return 0, 0, false
	}
	currentLine := 1
	start := 0
	for i, b := range src {
		if currentLine == line {
			end := i
			for end < len(src) && src[end] != '\n' {
				end++
			}
			return uint(start), uint(end), true
		}
		if b == '\n' {
			currentLine++
			start = i + 1
		}
	}
	if currentLine == line {
		return uint(start), uint(len(src)), true
	}
	return 0, 0, false
}

func byteRangeForLSPRange(src []byte, line, character, endLine, endCharacter int) (uint, uint, bool) {
	start, ok := byteOffsetForLSPPosition(src, line, character)
	if !ok {
		return 0, 0, false
	}
	end := start + 1
	if endLine > 0 && endCharacter > 0 {
		if resolvedEnd, ok := byteOffsetForLSPPosition(src, endLine, endCharacter); ok && resolvedEnd > start {
			end = resolvedEnd
		}
	}
	if end > uint(len(src)) {
		end = uint(len(src))
	}
	return start, end, true
}

func byteOffsetForLSPPosition(src []byte, line, character int) (uint, bool) {
	start, end, ok := lineByteRange(src, line)
	if !ok || character <= 0 {
		return 0, false
	}
	targetUnits := character - 1
	currentUnits := 0
	for offset := int(start); offset < int(end); {
		if currentUnits >= targetUnits {
			return uint(offset), true
		}
		r, size := utf8.DecodeRune(src[offset:int(end)])
		if r == utf8.RuneError && size == 0 {
			return uint(offset), true
		}
		if r == utf8.RuneError && size == 1 {
			offset++
			currentUnits++
			continue
		}
		offset += size
		currentUnits += lspCharacterWidth(r)
	}
	if currentUnits == targetUnits {
		return end, true
	}
	return 0, false
}

func lspCharacterForByteColumn(src []byte, row uint, byteColumn uint) int {
	line := int(row) + 1
	start, end, ok := lineByteRange(src, line)
	if !ok {
		return int(byteColumn) + 1
	}
	target := start + byteColumn
	if target > end {
		target = end
	}
	units := 0
	for offset := int(start); offset < int(target); {
		r, size := utf8.DecodeRune(src[offset:int(target)])
		if r == utf8.RuneError && size == 0 {
			break
		}
		if r == utf8.RuneError && size == 1 {
			offset++
			units++
			continue
		}
		offset += size
		units += lspCharacterWidth(r)
	}
	return units + 1
}

func lspCharacterForByteOffset(src []byte, byteOffset uint32) int {
	offset := uint(byteOffset)
	if offset > uint(len(src)) {
		offset = uint(len(src))
	}
	row := uint(0)
	lineStart := uint(0)
	for i := uint(0); i < offset; i++ {
		if src[i] == '\n' {
			row++
			lineStart = i + 1
		}
	}
	return lspCharacterForByteColumn(src, row, offset-lineStart)
}

func lspCharacterWidth(r rune) int {
	if r > 0xffff {
		return 2
	}
	return 1
}

func findIdentifierOccurrences(line []byte, target string) []uint {
	if len(line) == 0 || target == "" {
		return nil
	}
	targetBytes := []byte(target)
	var occurrences []uint
	for i := 0; i+len(targetBytes) <= len(line); i++ {
		if !bytesEqual(line[i:i+len(targetBytes)], targetBytes) {
			continue
		}
		if (i > 0 && isJSIdentifierPart(line[i-1])) || (i+len(targetBytes) < len(line) && isJSIdentifierPart(line[i+len(targetBytes)])) {
			continue
		}
		occurrences = append(occurrences, uint(i))
	}
	return occurrences
}

func normalizeJSFamilySourceForParse(src []byte) []byte {
	if len(src) == 0 {
		return src
	}
	out := append([]byte(nil), src...)
	changed := false
	for i := 0; i+len("async") <= len(out); i++ {
		if !bytesEqual(out[i:i+len("async")], []byte("async")) {
			continue
		}
		if (i > 0 && isJSIdentifierPart(out[i-1])) || (i+len("async") < len(out) && isJSIdentifierPart(out[i+len("async")])) {
			continue
		}
		j := i + len("async")
		for j < len(out) && isJSWhitespace(out[j]) {
			j++
		}
		if j >= len(out) || out[j] != '(' {
			continue
		}
		for k := i; k < i+len("async"); k++ {
			out[k] = ' '
		}
		changed = true
	}
	if !changed {
		return src
	}
	return out
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isJSIdentifierPart(ch byte) bool {
	return ch == '_' || ch == '$' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}

func isJSIdentifierStart(ch byte) bool {
	return ch == '_' || ch == '$' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isJSWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func sourceInCommentAt(src []byte, byteOffset uint) bool {
	state := sourceLexStateAt(src, byteOffset)
	return state.state == lexLineComment || state.state == lexBlockComment
}

func sourceInNonCodeAt(src []byte, byteOffset uint) bool {
	state := sourceLexStateAt(src, byteOffset)
	return state.state != lexNormal || state.regex
}

// LineStartsInNonCodeWithParsed は指定行の開始位置がコメント、文字列、テンプレート、正規表現 literal 内かを返す。
func LineStartsInNonCodeWithParsed(parsed *ParsedFile, line int) bool {
	if parsed == nil {
		return false
	}
	start, _, ok := lineByteRange(parsed.src, line)
	if !ok {
		return false
	}
	return sourceInNonCodeAt(parsed.src, start)
}

type sourceLexState struct {
	state      lexicalState
	regex      bool
	regexClass bool
	escaped    bool
}

func sourceLexStateAt(src []byte, byteOffset uint) sourceLexState {
	if byteOffset > uint(len(src)) {
		byteOffset = uint(len(src))
	}
	state := sourceLexState{state: lexNormal}
	for i := uint(0); i < byteOffset; i++ {
		ch := src[i]
		next := byte(0)
		if i+1 < uint(len(src)) {
			next = src[i+1]
		}

		if state.regex {
			switch {
			case state.escaped:
				state.escaped = false
			case ch == '\\':
				state.escaped = true
			case ch == '[':
				state.regexClass = true
			case ch == ']':
				state.regexClass = false
			case ch == '/' && !state.regexClass:
				state.regex = false
			case ch == '\n':
				state.regex = false
				state.regexClass = false
				state.escaped = false
			}
			continue
		}

		switch state.state {
		case lexLineComment:
			if ch == '\n' {
				state.state = lexNormal
			}
			continue
		case lexBlockComment:
			if ch == '*' && next == '/' {
				i++
				state.state = lexNormal
			}
			continue
		case lexSingleString:
			if state.escaped {
				state.escaped = false
				continue
			}
			if ch == '\\' {
				state.escaped = true
				continue
			}
			if ch == '\'' || ch == '\n' {
				state.state = lexNormal
			}
			continue
		case lexDoubleString:
			if state.escaped {
				state.escaped = false
				continue
			}
			if ch == '\\' {
				state.escaped = true
				continue
			}
			if ch == '"' || ch == '\n' {
				state.state = lexNormal
			}
			continue
		case lexTemplateString:
			if state.escaped {
				state.escaped = false
				continue
			}
			if ch == '\\' {
				state.escaped = true
				continue
			}
			if ch == '`' {
				state.state = lexNormal
			}
			continue
		}
		switch {
		case ch == '/' && next == '/':
			state.state = lexLineComment
			i++
		case ch == '/' && next == '*':
			state.state = lexBlockComment
			i++
		case ch == '/' && sourceSlashStartsRegex(src, i):
			state.regex = true
			state.regexClass = false
			state.escaped = false
		case ch == '\'':
			state.state = lexSingleString
		case ch == '"':
			state.state = lexDoubleString
		case ch == '`':
			state.state = lexTemplateString
		}
	}
	return state
}

func sourceSlashStartsRegex(src []byte, byteOffset uint) bool {
	prev, ok := previousNonSpaceByte(src, byteOffset)
	if !ok {
		return true
	}
	switch prev {
	case '(', '[', '{', '=', ':', ',', ';', '!', '&', '|', '?', '+', '-', '*', '~', '^', '<', '>':
		return true
	}
	prefix := strings.TrimSpace(string(src[:byteOffset]))
	for _, keyword := range []string{"return", "throw", "case", "delete", "typeof", "void", "yield"} {
		if sourcePrefixEndsWithKeyword(prefix, keyword) {
			return true
		}
	}
	return false
}

func sourcePrefixEndsWithKeyword(prefix string, keyword string) bool {
	if !strings.HasSuffix(prefix, keyword) {
		return false
	}
	before := len(prefix) - len(keyword) - 1
	return before < 0 || !isJSIdentifierPart(prefix[before])
}

func previousNonSpaceByte(src []byte, byteOffset uint) (byte, bool) {
	if byteOffset > uint(len(src)) {
		byteOffset = uint(len(src))
	}
	for i := int(byteOffset) - 1; i >= 0; i-- {
		if isJSWhitespace(src[i]) {
			continue
		}
		return src[i], true
	}
	return 0, false
}
