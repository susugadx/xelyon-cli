package tui

import (
	"unicode"

	termtext "github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

func (m *Model) moveCursorTo(line int) {
	if len(m.rawLines) == 0 {
		m.cursorLine = 0
		m.cursorCol = 0
		return
	}
	m.cursorLine = line
	m.clampCursorLine()
	m.ensureCursorVisible()
	m.chromeDirty = true
}

func (m *Model) moveCursorCol(delta int) {
	if len(m.rawLines) == 0 {
		m.cursorCol = 0
		return
	}
	if delta > 0 {
		for range delta {
			m.cursorCol = m.nextClusterStart(m.cursorLine, m.cursorCol)
		}
	} else if delta < 0 {
		for range -delta {
			m.cursorCol = m.prevClusterStart(m.cursorLine, m.cursorCol)
		}
	}
	m.clampCursorCol()
	m.chromeDirty = true
}

func (m *Model) moveCursorToLineStart(firstNonBlank bool, count int) {
	if len(m.rawLines) == 0 {
		m.cursorLine = 0
		m.cursorCol = 0
		return
	}
	if count < 1 {
		count = 1
	}
	targetLine := min(m.cursorLine+count-1, len(m.rawLines)-1)
	m.cursorLine = targetLine
	line := termtext.StripANSI(m.rawLines[targetLine])
	m.cursorCol = 0
	if firstNonBlank {
		for _, r := range line {
			if !unicode.IsSpace(r) {
				break
			}
			m.cursorCol += termtext.RuneWidth(r)
		}
	}
	m.clampCursorCol()
	m.ensureCursorVisible()
	m.chromeDirty = true
}

func (m *Model) moveCursorToLineEnd(count int) {
	if len(m.rawLines) == 0 {
		m.cursorLine = 0
		m.cursorCol = 0
		return
	}
	if count < 1 {
		count = 1
	}
	targetLine := min(m.cursorLine+count-1, len(m.rawLines)-1)
	m.cursorLine = targetLine
	m.cursorCol = m.maxCursorColForLine(targetLine)
	m.ensureCursorVisible()
	m.chromeDirty = true
}
