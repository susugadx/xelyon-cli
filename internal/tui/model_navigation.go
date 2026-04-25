package tui

import termtext "github.com/susugadx/xelyon-cli/internal/tui/termtext"

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
		text := termtext.StripANSI(m.rawLines[lineIdx])
		runes := []rune(text)
		start := 0
		if lineIdx == line {
			start = termtext.DisplayColToRuneIndex(text, col)
			if start < len(runes) {
				inWord = termtext.IsWordRune(runes[start])
				start++
			}
		}
		for idx := start; idx < len(runes); idx++ {
			if termtext.IsWordRune(runes[idx]) {
				if !inWord || idx == 0 || !termtext.IsWordRune(runes[idx-1]) {
					return lineIdx, termtext.RuneIndexToDisplayCol(text, idx)
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
		text := termtext.StripANSI(m.rawLines[lineIdx])
		runes := []rune(text)
		if len(runes) == 0 {
			continue
		}
		idx := len(runes) - 1
		if lineIdx == line {
			idx = termtext.DisplayColToRuneIndex(text, col)
			if idx >= len(runes) {
				idx = len(runes) - 1
			}
		}
		for idx >= 0 && !termtext.IsWordRune(runes[idx]) {
			idx--
		}
		for idx >= 0 && termtext.IsWordRune(runes[idx]) {
			if idx == 0 || !termtext.IsWordRune(runes[idx-1]) {
				return lineIdx, termtext.RuneIndexToDisplayCol(text, idx)
			}
			idx--
		}
	}
	return 0, 0
}

func (m Model) findWordEnd(line, col int) (int, int) {
	inWord := false
	for lineIdx := line; lineIdx < len(m.rawLines); lineIdx++ {
		text := termtext.StripANSI(m.rawLines[lineIdx])
		runes := []rune(text)
		start := 0
		if lineIdx == line {
			start = termtext.DisplayColToRuneIndex(text, col)
			if start < len(runes) && termtext.IsWordRune(runes[start]) {
				inWord = true
			}
		}
		for idx := start; idx < len(runes); idx++ {
			if termtext.IsWordRune(runes[idx]) {
				inWord = true
				if idx == len(runes)-1 || !termtext.IsWordRune(runes[idx+1]) {
					return lineIdx, termtext.RuneIndexToDisplayCol(text, idx)
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
