package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

// setTransientStatus は一時通知メッセージを設定する。
func (m *Model) setTransientStatus(text string) {
	m.transientStatus = text
	m.transientStatusUntil = time.Now().Add(2 * time.Second)
	m.chromeDirty = true
}

func (m *Model) resetNavPending() {
	m.gPressed = false
	m.pendingCount = 0
	m.yPressed = false
}

func (m *Model) clearVisualSelection() {
	m.visualMode = visualModeOff
	m.visualStart = visualPosition{line: -1, col: -1}
	m.yPressed = false
}

func (m *Model) afterViewportScroll() {
	if m.navigationMode && m.focusedBlock < 0 && m.visualMode == visualModeOff {
		m.clampCursorToViewport()
	}
	if m.vp.atBottom() {
		m.newOutput = false
	}
	m.chromeDirty = true
}

func (m *Model) clampCursorLine() {
	if len(m.rawLines) == 0 {
		m.cursorLine = 0
		m.cursorCol = 0
		return
	}
	if m.cursorLine < 0 {
		m.cursorLine = 0
	}
	if m.cursorLine >= len(m.rawLines) {
		m.cursorLine = len(m.rawLines) - 1
	}
	m.clampCursorCol()
}

func (m *Model) syncCursorToViewportTop() {
	if len(m.rawLines) == 0 {
		m.cursorLine = 0
		m.cursorCol = 0
		return
	}
	if m.layout != nil && m.vp.yOffset < len(m.layout.Rows) {
		m.cursorLine = m.layout.Rows[m.vp.yOffset].RawLineIdx
		m.cursorCol = m.layout.GetRawColumnForVisualRow(m.vp.yOffset)
	} else {
		m.cursorLine = 0
		m.cursorCol = 0
	}
	m.clampCursorCol()
}

func (m *Model) ensureCursorVisible() {
	m.clampCursorLine()
	if m.vp.height <= 0 {
		return
	}

	cursorRowIdx, _ := -1, -1
	if m.layout != nil {
		cursorRowIdx, _ = m.layout.GetVisualCursor(m.cursorLine, m.cursorCol)
	}
	if cursorRowIdx < 0 {
		return
	}

	if cursorRowIdx < m.vp.yOffset {
		m.vp.yOffset = cursorRowIdx
	}
	if cursorRowIdx >= m.vp.yOffset+m.vp.height {
		m.vp.yOffset = cursorRowIdx - m.vp.height + 1
	}
	if m.vp.yOffset > m.vp.maxYOffset() {
		m.vp.yOffset = m.vp.maxYOffset()
	}
	if m.vp.yOffset < 0 {
		m.vp.yOffset = 0
	}
}

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
	m.cursorCol += delta
	m.clampCursorCol()
	m.chromeDirty = true
}

func (m *Model) consumePendingCountOr(fallback int) int {
	if m.pendingCount > 0 {
		count := m.pendingCount
		m.pendingCount = 0
		return count
	}
	return fallback
}

func (m *Model) clampCursorCol() {
	maxCol := m.maxCursorColForLine(m.cursorLine)
	if m.cursorCol < 0 {
		m.cursorCol = 0
	}
	if m.cursorCol > maxCol {
		m.cursorCol = maxCol
	}
}

func (m Model) maxCursorColForLine(line int) int {
	if line < 0 || line >= len(m.rawLines) {
		return 0
	}
	width := lipgloss.Width(stripANSI(m.rawLines[line]))
	if width <= 0 {
		return 0
	}
	return width - 1
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
	line := stripANSI(m.rawLines[targetLine])
	m.cursorCol = 0
	if firstNonBlank {
		for _, r := range line {
			if !unicode.IsSpace(r) {
				break
			}
			m.cursorCol += runeWidth(r)
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

func (m Model) lineSelectionRange() (start, end int, ok bool) {
	if m.visualMode == visualModeOff || m.visualStart.line < 0 {
		return 0, 0, false
	}
	start = min(m.visualStart.line, m.cursorLine)
	end = max(m.visualStart.line, m.cursorLine)
	return start, end, true
}

func (m Model) normalizedCharSelection() (start, end visualPosition, ok bool) {
	if m.visualMode != visualModeChar || m.visualStart.line < 0 {
		return visualPosition{}, visualPosition{}, false
	}
	start = m.visualStart
	end = visualPosition{line: m.cursorLine, col: m.cursorCol}
	if start.line > end.line || (start.line == end.line && start.col > end.col) {
		start, end = end, start
	}
	return start, end, true
}

func (m *Model) copyCursorLine() {
	if err := m.copyRawRangePlain(m.cursorLine, m.cursorLine); err != nil {
		m.setTransientStatus("Copy failed: " + err.Error())
		return
	}
	m.setTransientStatus("✅ Copied 1 line")
}

func (m *Model) copyVisualSelection() {
	switch m.visualMode {
	case visualModeChar:
		text, lines := m.copyCharVisualSelectionText()
		if err := m.agent.CopyText(text); err != nil {
			m.setTransientStatus("Copy failed: " + err.Error())
			return
		}
		m.clearVisualSelection()
		m.setTransientStatus("✅ Copied " + lineLabel(lines))
	case visualModeLine:
		start, end, ok := m.lineSelectionRange()
		if !ok {
			return
		}
		if err := m.copyRawRangePlain(start, end); err != nil {
			m.setTransientStatus("Copy failed: " + err.Error())
			return
		}
		m.clearVisualSelection()
		m.setTransientStatus("✅ Copied " + lineLabel(end-start+1))
	}
}

func (m Model) copyCharVisualSelectionText() (string, int) {
	start, end, ok := m.normalizedCharSelection()
	if !ok || len(m.rawLines) == 0 {
		return "", 0
	}

	var result strings.Builder
	for i := start.line; i <= end.line; i++ {
		line := stripANSI(m.rawLines[i])
		runes := []rune(line)
		from := 0
		to := len(runes)

		if i == start.line {
			from = displayColToRuneIndex(line, start.col)
		}
		if i == end.line {
			to = displayColToRuneIndexAfter(line, end.col)
		}
		if from > len(runes) {
			from = len(runes)
		}
		if to > len(runes) {
			to = len(runes)
		}
		if from > to {
			from = to
		}

		if i > start.line {
			result.WriteByte('\n')
		}
		result.WriteString(string(runes[from:to]))
	}

	return result.String(), end.line - start.line + 1
}

func (m *Model) copyDefaultSelectionTarget() {
	if msg, err := m.agent.CopyLastOutput(); err == nil {
		m.setTransientStatus("✅ " + msg)
	} else {
		m.setTransientStatus("Copy failed: " + err.Error())
	}
}

func (m Model) copyRawRangePlain(start, end int) error {
	if len(m.rawLines) == 0 {
		return fmt.Errorf("no lines to copy")
	}
	start = max(0, start)
	end = min(len(m.rawLines)-1, end)
	if start > end {
		return nil
	}

	lines := make([]string, 0, end-start+1)
	for _, line := range m.rawLines[start : end+1] {
		lines = append(lines, stripANSI(line))
	}
	return m.agent.CopyText(strings.Join(lines, "\n"))
}

func lineLabel(n int) string {
	if n == 1 {
		return "1 line"
	}
	return strconv.Itoa(n) + " lines"
}
