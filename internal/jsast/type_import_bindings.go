package jsast

// TypeImportBindingsWithParsed は type-only import の local binding を抽出する。
func TypeImportBindingsWithParsed(parsed *ParsedFile) []ImportBinding {
	if parsed == nil {
		return nil
	}
	return typeImportBindingsFromSource(parsed.src)
}

func typeImportBindingsFromSource(src []byte) []ImportBinding {
	var bindings []ImportBinding
	for offset := 0; offset < len(src); {
		importStart, ok := nextImportStatementStart(src, offset)
		if !ok {
			break
		}
		importEnd := importStatementEnd(src, importStart)
		bindings = append(bindings, typeImportBindingsFromStatement(src[importStart:importEnd], importStart, src)...)
		offset = importEnd
	}
	return bindings
}

func nextImportStatementStart(src []byte, offset int) (int, bool) {
	state := lexNormal
	escaped := false
	for i := offset; i < len(src); i++ {
		ch := src[i]
		next := byte(0)
		if i+1 < len(src) {
			next = src[i+1]
		}
		state, escaped = advanceImportBindingLexState(state, escaped, ch, next)
		if state != lexNormal {
			if (ch == '/' && (next == '/' || next == '*')) || (ch == '*' && next == '/') {
				i++
			}
			continue
		}
		if hasImportBindingKeywordAtBytes(src, i, "import") && importStatementBoundaryBefore(src, i) {
			return i, true
		}
	}
	return 0, false
}

func importStatementEnd(src []byte, start int) int {
	state := lexNormal
	escaped := false
	braceDepth := 0
	for i := start; i < len(src); i++ {
		ch := src[i]
		next := byte(0)
		if i+1 < len(src) {
			next = src[i+1]
		}
		if state == lexNormal {
			switch ch {
			case '{':
				braceDepth++
			case '}':
				if braceDepth > 0 {
					braceDepth--
				}
			case ';':
				if braceDepth == 0 {
					return i + 1
				}
			case '\n', '\r':
				if braceDepth == 0 {
					return i
				}
			}
		}
		state, escaped = advanceImportBindingLexState(state, escaped, ch, next)
		if (ch == '/' && (next == '/' || next == '*')) || (ch == '*' && next == '/') {
			i++
		}
	}
	return len(src)
}

func typeImportBindingsFromStatement(stmt []byte, baseOffset int, fullSrc []byte) []ImportBinding {
	source := importBindingStatementSource(stmt)
	if source == "" {
		return nil
	}
	statementStartLine := lineForByteOffset(fullSrc, baseOffset)
	statementEndLine := lineForByteOffset(fullSrc, baseOffset+len(stmt))
	i := skipImportBindingWhitespaceAndCommentsBytes(stmt, 0)
	if !hasImportBindingKeywordAtBytes(stmt, i, "import") {
		return nil
	}
	i = skipImportBindingWhitespaceAndCommentsBytes(stmt, i+len("import"))
	statementTypeOnly := false
	if keyword, ok := importBindingTypeKeywordAtBytes(stmt, i); ok {
		statementTypeOnly = true
		i = skipImportBindingWhitespaceAndCommentsBytes(stmt, i+len(keyword))
	}

	var bindings []ImportBinding
	if statementTypeOnly {
		if binding, ok := defaultTypeImportBindingFromStatement(stmt, i, baseOffset, fullSrc, source); ok {
			bindings = append(bindings, binding.withStatementLines(statementStartLine, statementEndLine))
		}
	}
	if start, end, ok := importBindingNamedListRange(stmt); ok {
		for _, binding := range typeImportBindingsFromNamedList(stmt[start:end], baseOffset+start, fullSrc, source, statementTypeOnly) {
			bindings = append(bindings, binding.withStatementLines(statementStartLine, statementEndLine))
		}
	}
	return bindings
}

func defaultTypeImportBindingFromStatement(stmt []byte, offset int, baseOffset int, fullSrc []byte, source string) (ImportBinding, bool) {
	if offset >= len(stmt) || stmt[offset] == '{' || stmt[offset] == '*' {
		return ImportBinding{}, false
	}
	name, start, end, ok := readImportBindingIdentifier(stmt, offset)
	if !ok || name == "" {
		return ImportBinding{}, false
	}
	return ImportBinding{
		Kind:           ImportBindingType,
		Imported:       "default",
		Local:          name,
		Source:         source,
		Line:           lineForByteOffset(fullSrc, baseOffset+start),
		localStartByte: uint32(baseOffset + start),
		localEndByte:   uint32(baseOffset + end),
	}, true
}

func typeImportBindingsFromNamedList(list []byte, baseOffset int, fullSrc []byte, source string, statementTypeOnly bool) []ImportBinding {
	var bindings []ImportBinding
	for _, spec := range splitImportBindingSpecifiers(list) {
		if binding, ok := typeImportBindingFromSpecifier(list[spec.start:spec.end], baseOffset+spec.start, fullSrc, source, statementTypeOnly); ok {
			bindings = append(bindings, binding)
		}
	}
	return bindings
}

type importBindingSpecifierRange struct {
	start int
	end   int
}

func splitImportBindingSpecifiers(list []byte) []importBindingSpecifierRange {
	var ranges []importBindingSpecifierRange
	state := lexNormal
	escaped := false
	start := 0
	for i := 0; i < len(list); i++ {
		ch := list[i]
		next := byte(0)
		if i+1 < len(list) {
			next = list[i+1]
		}
		if state == lexNormal && ch == ',' {
			ranges = appendImportBindingSpecifierRange(ranges, list, start, i)
			start = i + 1
		}
		state, escaped = advanceImportBindingLexState(state, escaped, ch, next)
		if (ch == '/' && (next == '/' || next == '*')) || (ch == '*' && next == '/') {
			i++
		}
	}
	return appendImportBindingSpecifierRange(ranges, list, start, len(list))
}

func appendImportBindingSpecifierRange(ranges []importBindingSpecifierRange, src []byte, start int, end int) []importBindingSpecifierRange {
	start = skipImportBindingWhitespaceAndCommentsBytes(src, start)
	end = trimImportBindingWhitespaceAndCommentsRight(src, end)
	if start < end {
		ranges = append(ranges, importBindingSpecifierRange{start: start, end: end})
	}
	return ranges
}

func typeImportBindingFromSpecifier(spec []byte, baseOffset int, fullSrc []byte, source string, statementTypeOnly bool) (ImportBinding, bool) {
	i := skipImportBindingWhitespaceAndCommentsBytes(spec, 0)
	if !statementTypeOnly {
		keyword, ok := importBindingTypeKeywordAtBytes(spec, i)
		if !ok {
			return ImportBinding{}, false
		}
		i = skipImportBindingWhitespaceAndCommentsBytes(spec, i+len(keyword))
		if hasImportBindingKeywordAtBytes(spec, i, "as") || i >= len(spec) {
			return ImportBinding{}, false
		}
	}
	imported, importedStart, importedEnd, ok := readImportBindingIdentifier(spec, i)
	if !ok || imported == "" {
		return ImportBinding{}, false
	}
	local := imported
	localStart := importedStart
	localEnd := importedEnd
	if alias, aliasStart, aliasEnd, ok := importBindingAliasIdentifier(spec, importedEnd); ok {
		local = alias
		localStart = aliasStart
		localEnd = aliasEnd
	}
	return ImportBinding{
		Kind:           ImportBindingType,
		Imported:       imported,
		Local:          local,
		Source:         source,
		Line:           lineForByteOffset(fullSrc, baseOffset+localStart),
		localStartByte: uint32(baseOffset + localStart),
		localEndByte:   uint32(baseOffset + localEnd),
	}, true
}

func importBindingAliasIdentifier(spec []byte, offset int) (string, int, int, bool) {
	i := skipImportBindingWhitespaceAndCommentsBytes(spec, offset)
	if !hasImportBindingKeywordAtBytes(spec, i, "as") {
		return "", 0, 0, false
	}
	return readImportBindingIdentifier(spec, i+len("as"))
}

func readImportBindingIdentifier(src []byte, offset int) (string, int, int, bool) {
	i := skipImportBindingWhitespaceAndCommentsBytes(src, offset)
	if i >= len(src) || !isJSIdentifierStart(src[i]) {
		return "", 0, 0, false
	}
	start := i
	i++
	for i < len(src) && isJSIdentifierPart(src[i]) {
		i++
	}
	return string(src[start:i]), start, i, true
}

func importBindingNamedListRange(stmt []byte) (int, int, bool) {
	state := lexNormal
	escaped := false
	start := -1
	depth := 0
	for i := 0; i < len(stmt); i++ {
		ch := stmt[i]
		next := byte(0)
		if i+1 < len(stmt) {
			next = stmt[i+1]
		}
		if state == lexNormal {
			switch ch {
			case '{':
				if depth == 0 {
					start = i + 1
				}
				depth++
			case '}':
				if depth > 0 {
					depth--
					if depth == 0 && start >= 0 {
						return start, i, true
					}
				}
			}
		}
		state, escaped = advanceImportBindingLexState(state, escaped, ch, next)
		if (ch == '/' && (next == '/' || next == '*')) || (ch == '*' && next == '/') {
			i++
		}
	}
	return 0, 0, false
}

func importBindingStatementSource(stmt []byte) string {
	state := lexNormal
	escaped := false
	var last string
	for i := 0; i < len(stmt); i++ {
		ch := stmt[i]
		next := byte(0)
		if i+1 < len(stmt) {
			next = stmt[i+1]
		}
		if state == lexNormal && (ch == '\'' || ch == '"') {
			if literal, end, ok := importBindingStringLiteral(stmt, i); ok {
				last = literal
				i = end - 1
				continue
			}
		}
		state, escaped = advanceImportBindingLexState(state, escaped, ch, next)
		if (ch == '/' && (next == '/' || next == '*')) || (ch == '*' && next == '/') {
			i++
		}
	}
	return last
}

func importBindingStringLiteral(src []byte, start int) (string, int, bool) {
	quote := src[start]
	escaped := false
	for i := start + 1; i < len(src); i++ {
		ch := src[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == quote {
			return string(src[start+1 : i]), i + 1, true
		}
	}
	return "", start, false
}

func importBindingTypeKeywordAtBytes(src []byte, offset int) (string, bool) {
	for _, keyword := range []string{"typeof", "type"} {
		if hasImportBindingKeywordAtBytes(src, offset, keyword) {
			return keyword, true
		}
	}
	return "", false
}

func skipImportBindingWhitespaceAndCommentsBytes(src []byte, offset int) int {
	i := offset
	for i < len(src) {
		switch {
		case isJSWhitespace(src[i]):
			i++
		case i+1 < len(src) && src[i] == '/' && src[i+1] == '*':
			i += 2
			for i+1 < len(src) && (src[i] != '*' || src[i+1] != '/') {
				i++
			}
			if i+1 < len(src) {
				i += 2
			}
		case i+1 < len(src) && src[i] == '/' && src[i+1] == '/':
			i += 2
			for i < len(src) && src[i] != '\n' && src[i] != '\r' {
				i++
			}
		default:
			return i
		}
	}
	return i
}

func trimImportBindingWhitespaceAndCommentsRight(src []byte, end int) int {
	for end > 0 && isJSWhitespace(src[end-1]) {
		end--
	}
	return end
}

func hasImportBindingKeywordAtBytes(src []byte, offset int, keyword string) bool {
	if offset < 0 || offset+len(keyword) > len(src) || string(src[offset:offset+len(keyword)]) != keyword {
		return false
	}
	beforeOK := offset == 0 || !isJSIdentifierPart(src[offset-1])
	after := offset + len(keyword)
	afterOK := after >= len(src) || !isJSIdentifierPart(src[after])
	return beforeOK && afterOK
}

func importStatementBoundaryBefore(src []byte, offset int) bool {
	for i := offset - 1; i >= 0; i-- {
		if isJSWhitespace(src[i]) {
			if src[i] == '\n' || src[i] == '\r' {
				return true
			}
			continue
		}
		switch src[i] {
		case ';', '{', '}':
			return true
		default:
			return false
		}
	}
	return true
}

func advanceImportBindingLexState(state lexicalState, escaped bool, ch byte, next byte) (lexicalState, bool) {
	switch state {
	case lexLineComment:
		if ch == '\n' || ch == '\r' {
			return lexNormal, false
		}
		return state, false
	case lexBlockComment:
		if ch == '*' && next == '/' {
			return lexNormal, false
		}
		return state, false
	case lexSingleString:
		return advanceImportBindingStringState(state, escaped, ch, '\'')
	case lexDoubleString:
		return advanceImportBindingStringState(state, escaped, ch, '"')
	case lexTemplateString:
		return advanceImportBindingStringState(state, escaped, ch, '`')
	default:
		switch {
		case ch == '/' && next == '/':
			return lexLineComment, false
		case ch == '/' && next == '*':
			return lexBlockComment, false
		case ch == '\'':
			return lexSingleString, false
		case ch == '"':
			return lexDoubleString, false
		case ch == '`':
			return lexTemplateString, false
		default:
			return lexNormal, false
		}
	}
}

func advanceImportBindingStringState(state lexicalState, escaped bool, ch byte, quote byte) (lexicalState, bool) {
	if escaped {
		return state, false
	}
	if ch == '\\' {
		return state, true
	}
	if ch == quote {
		return lexNormal, false
	}
	return state, false
}

func lineForByteOffset(src []byte, offset int) int {
	if offset < 0 {
		offset = 0
	}
	if offset > len(src) {
		offset = len(src)
	}
	line := 1
	for i := 0; i < offset; i++ {
		if src[i] == '\n' {
			line++
		}
	}
	return line
}
