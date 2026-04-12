package tui

func (m *Model) moveCursorToNextWordStart(count int) {
	if len(m.rawLines) == 0 {
		return
	}
	line, col := m.cursorLine, m.cursorCol
	for range count {
		line, col = m.findNextWordStart(line, col)
	}
	m.cursorLine = line
	m.cursorCol = col
	m.ensureCursorVisible()
	m.chromeDirty = true
}

func (m *Model) moveCursorToPrevWordStart(count int) {
	if len(m.rawLines) == 0 {
		return
	}
	line, col := m.cursorLine, m.cursorCol
	for range count {
		line, col = m.findPrevWordStart(line, col)
	}
	m.cursorLine = line
	m.cursorCol = col
	m.ensureCursorVisible()
	m.chromeDirty = true
}

func (m *Model) moveCursorToWordEnd(count int) {
	if len(m.rawLines) == 0 {
		return
	}
	line, col := m.cursorLine, m.cursorCol
	for range count {
		line, col = m.findWordEnd(line, col)
	}
	m.cursorLine = line
	m.cursorCol = col
	m.ensureCursorVisible()
	m.chromeDirty = true
}

func (m Model) findNextWordStart(line, col int) (int, int) {
	inWord := false
	for lineIdx := line; lineIdx < len(m.rawLines); lineIdx++ {
		text := stripANSI(m.rawLines[lineIdx])
		runes := []rune(text)
		start := 0
		if lineIdx == line {
			start = displayColToRuneIndex(text, col)
			if start < len(runes) {
				inWord = isWordRune(runes[start])
				start++
			}
		}
		for idx := start; idx < len(runes); idx++ {
			if isWordRune(runes[idx]) {
				if !inWord || idx == 0 || !isWordRune(runes[idx-1]) {
					return lineIdx, runeIndexToDisplayCol(text, idx)
				}
			} else {
				inWord = false
			}
		}
		inWord = false
	}
	lastLine := len(m.rawLines) - 1
	return lastLine, m.maxCursorColForLine(lastLine)
}

func (m Model) findPrevWordStart(line, col int) (int, int) {
	for lineIdx := line; lineIdx >= 0; lineIdx-- {
		text := stripANSI(m.rawLines[lineIdx])
		runes := []rune(text)
		if len(runes) == 0 {
			continue
		}
		idx := len(runes) - 1
		if lineIdx == line {
			idx = displayColToRuneIndex(text, col)
			if idx >= len(runes) {
				idx = len(runes) - 1
			}
		}
		for idx >= 0 && !isWordRune(runes[idx]) {
			idx--
		}
		for idx >= 0 && isWordRune(runes[idx]) {
			if idx == 0 || !isWordRune(runes[idx-1]) {
				return lineIdx, runeIndexToDisplayCol(text, idx)
			}
			idx--
		}
	}
	return 0, 0
}

func (m Model) findWordEnd(line, col int) (int, int) {
	inWord := false
	for lineIdx := line; lineIdx < len(m.rawLines); lineIdx++ {
		text := stripANSI(m.rawLines[lineIdx])
		runes := []rune(text)
		start := 0
		if lineIdx == line {
			start = displayColToRuneIndex(text, col)
			if start < len(runes) && isWordRune(runes[start]) {
				inWord = true
			}
		}
		for idx := start; idx < len(runes); idx++ {
			if isWordRune(runes[idx]) {
				inWord = true
				if idx == len(runes)-1 || !isWordRune(runes[idx+1]) {
					return lineIdx, runeIndexToDisplayCol(text, idx)
				}
				continue
			}
			if inWord {
				inWord = false
			}
		}
		inWord = false
	}
	lastLine := len(m.rawLines) - 1
	return lastLine, m.maxCursorColForLine(lastLine)
}
