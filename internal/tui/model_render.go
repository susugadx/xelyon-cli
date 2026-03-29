package tui

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/rivo/uniseg"
)

const visualTabWidth = 4

func (m *Model) rebuildLayout() {
	m.layout = BuildLayout(m.rawLines, m.width)
}

func (m *Model) getVisualRowContents() []string {
	if m.layout == nil {
		return nil
	}
	res := make([]string, len(m.layout.Rows))
	for i, r := range m.layout.Rows {
		res[i] = r.Content
	}
	return res
}

// truncateWithANSI は ANSI エスケープを保持しつつ表示幅を制限する。
// CJK 全角文字（幅2）を正しくカウントする。
func truncateWithANSI(s string, maxWidth int) string {
	var result strings.Builder
	width := 0
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			result.WriteRune(r)
			continue
		}
		if inEscape {
			result.WriteRune(r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		w := runeWidth(r)
		if width+w > maxWidth {
			break
		}
		result.WriteRune(r)
		width += w
	}
	truncated := result.String()
	if strings.Contains(truncated, "\033[") && !strings.HasSuffix(truncated, "\033[0m") {
		truncated += "\033[0m"
	}
	return truncated
}

// runeWidth は文字の表示幅を返す（CJK 全角 = 2、それ以外 = 1）。
func runeWidth(r rune) int {
	if r == '\t' {
		return visualTabWidth
	}
	if r >= 0x1100 &&
		(r <= 0x115F || r == 0x2329 || r == 0x232A ||
			(r >= 0x2E80 && r <= 0x303E) ||
			(r >= 0x3040 && r <= 0x33BF) ||
			(r >= 0x3400 && r <= 0x4DBF) ||
			(r >= 0x4E00 && r <= 0xA4CF) ||
			(r >= 0xA960 && r <= 0xA97C) ||
			(r >= 0xAC00 && r <= 0xD7A3) ||
			(r >= 0xF900 && r <= 0xFAFF) ||
			(r >= 0xFE10 && r <= 0xFE6F) ||
			(r >= 0xFF01 && r <= 0xFF60) ||
			(r >= 0xFFE0 && r <= 0xFFE6) ||
			(r >= 0x1F000 && r <= 0x1FFFF) ||
			(r >= 0x20000 && r <= 0x3FFFF)) {
		return 2
	}
	return 1
}

func plainTextDisplayWidth(s string) int {
	return lipgloss.Width(strings.ReplaceAll(s, "\t", strings.Repeat(" ", visualTabWidth)))
}

func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

func displayColToRuneIndex(s string, col int) int {
	if col <= 0 {
		return 0
	}
	width := 0
	runeIdx := 0
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		cluster := gr.Str()
		next := width + plainTextDisplayWidth(cluster)
		if col < next {
			return runeIdx
		}
		width = next
		runeIdx += utf8.RuneCountInString(cluster)
	}
	return runeIdx
}

func displayColToRuneIndexAfter(s string, col int) int {
	if col < 0 {
		return 0
	}
	width := 0
	runeIdx := 0
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		cluster := gr.Str()
		clusterRunes := utf8.RuneCountInString(cluster)
		next := width + plainTextDisplayWidth(cluster)
		if col < next {
			return runeIdx + clusterRunes
		}
		width = next
		runeIdx += clusterRunes
	}
	return runeIdx
}

func runeIndexToDisplayCol(s string, idx int) int {
	if idx <= 0 {
		return 0
	}
	width := 0
	runeIdx := 0
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		cluster := gr.Str()
		clusterRunes := utf8.RuneCountInString(cluster)
		if runeIdx >= idx {
			break
		}
		width += plainTextDisplayWidth(cluster)
		runeIdx += clusterRunes
	}
	return width
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func stylePlainTextRange(s string, startCol, endCol int, bg string) string {
	if bg == "" || startCol >= endCol {
		return s
	}
	var result strings.Builder
	width := 0
	inRange := false
	for _, r := range s {
		rw := runeWidth(r)
		highlight := width < endCol && width+rw > startCol
		if highlight && !inRange {
			result.WriteString(bg)
			inRange = true
		} else if !highlight && inRange {
			result.WriteString("\033[0m")
			inRange = false
		}
		result.WriteRune(r)
		width += rw
	}
	if inRange {
		result.WriteString("\033[0m")
	}
	return result.String()
}

func stylePlainTextRangeWithCursor(s string, startCol, endCol int, rangeBg string, cursorCol int, cursorBg string, lineBg string) string {
	if s == "" {
		if cursorCol == 0 {
			if lineBg != "" {
				return lineBg + cursorBg + " " + "\033[0m"
			}
			return cursorBg + " " + "\033[0m"
		}
		if lineBg != "" {
			return lineBg + "\033[0m"
		}
		return ""
	}
	var result strings.Builder
	width := 0
	cursorPainted := false
	for _, r := range s {
		rw := runeWidth(r)
		if lineBg != "" {
			result.WriteString(lineBg)
		}
		if rangeBg != "" && startCol < endCol && width < endCol && width+rw > startCol {
			result.WriteString(rangeBg)
		}
		if !cursorPainted && cursorCol < width+rw {
			result.WriteString(cursorBg)
			cursorPainted = true
		}
		result.WriteRune(r)
		result.WriteString("\033[0m")
		width += rw
	}
	return result.String()
}

func (m Model) charSelectionColumnsForLine(line int) (startCol, endCol int, ok bool) {
	start, end, ok := m.normalizedCharSelection()
	if !ok || line < start.line || line > end.line {
		return 0, 0, false
	}

	plain := stripANSI(m.rawLines[line])
	lineWidth := plainTextDisplayWidth(plain)
	switch {
	case start.line == end.line:
		return start.col, min(lineWidth, end.col+1), true
	case line == start.line:
		return start.col, lineWidth, true
	case line == end.line:
		return 0, min(lineWidth, end.col+1), true
	default:
		return 0, lineWidth, true
	}
}

func decorateViewportLine(line string, width int, bg string) string {
	return fillANSITextWidth(line, width, bg)
}

func fitANSITextWidth(line string, width int) string {
	return fillANSITextWidth(line, width, "")
}

func fillANSITextWidth(line string, width int, bg string) string {
	if width <= 0 {
		return ""
	}
	line = truncateWithANSI(line, width)
	padding := max(0, width-lipgloss.Width(line))
	if bg == "" {
		return line + strings.Repeat(" ", padding)
	}
	line = strings.ReplaceAll(line, "\033[0m", "\033[0m"+bg)
	line = strings.ReplaceAll(line, "\033[m", "\033[m"+bg)
	line = strings.ReplaceAll(line, "\033[49m", "\033[49m"+bg)
	// Some subcomponents may clear background; re-apply before right padding.
	return bg + line + "\033[0m" + bg + strings.Repeat(" ", padding) + "\033[0m"
}

func sanitizeSingleLineANSI(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	inEscape := false
	spacePending := false

	emitSpace := func() {
		if b.Len() == 0 || spacePending {
			return
		}
		b.WriteByte(' ')
		spacePending = true
	}

	for _, r := range s {
		if inEscape {
			b.WriteRune(r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}

		if r == '\033' {
			inEscape = true
			b.WriteRune(r)
			continue
		}

		switch r {
		case '\r', '\n', '\t':
			emitSpace()
		default:
			if (r >= 0 && r < 0x20) || r == 0x7f {
				continue
			}
			b.WriteRune(r)
			spacePending = false
		}
	}

	return strings.TrimRight(b.String(), " ")
}

func (m Model) viewportView() string {
	if m.hasActiveMouseSelection() {
		return m.renderViewportWithMouseSelection()
	}
	if !m.navigationMode || m.focusedBlock >= 0 {
		return m.vp.view()
	}

	const cursorLineBg = "\033[48;5;236m"
	const cursorCharBg = "\033[48;5;255;38;5;16m"
	const visualBg = "\033[48;5;240m"
	const visualCursorBg = "\033[48;5;255;38;5;16m"

	visible := m.vp.visibleLines()
	var sb strings.Builder
	sb.Grow(m.vp.height * (m.vp.width + 1))

	for i := 0; i < m.vp.height; i++ {
		if i > 0 {
			sb.WriteByte('\n')
		}
		if i >= len(visible) {
			sb.WriteString(strings.Repeat(" ", max(0, m.vp.width)))
			continue
		}

		visIdx := m.vp.yOffset + i
		rawIdx := visIdx
		if m.layout != nil && visIdx < len(m.layout.Rows) {
			rawIdx = m.layout.Rows[visIdx].RawLineIdx
		}

		line := visible[i]

		switch m.visualMode {
		case visualModeChar:
			if startCol, endCol, ok := m.charSelectionColumnsForLine(rawIdx); ok {
				plain := stripANSI(line)

				// Local coordinate bounds
				localStartCol := startCol
				localEndCol := endCol
				if m.layout != nil && visIdx < len(m.layout.Rows) {
					if rawIdx > m.visualStart.line && rawIdx < m.cursorLine || rawIdx > m.cursorLine && rawIdx < m.visualStart.line {
						localStartCol = 0
						localEndCol = 9999
					} else {
						startVisRow, startVisCol := m.layout.GetVisualCursor(rawIdx, startCol)
						endVisRow, endVisCol := m.layout.GetVisualCursor(rawIdx, endCol)

						if endVisRow > startVisRow && endVisRow >= 0 && endVisRow < len(m.layout.Rows) {
							if endVisCol == m.layout.Rows[endVisRow].PrefixWidth {
								endVisRow--
								endVisCol = 9999
							}
						}

						if visIdx < startVisRow || visIdx > endVisRow {
							localStartCol = 0
							localEndCol = 0
						} else {
							if visIdx == startVisRow {
								localStartCol = startVisCol
							} else {
								localStartCol = 0
							}

							if visIdx == endVisRow {
								localEndCol = endVisCol
							} else {
								localEndCol = 9999
							}
						}
					}
				}

				styled := stylePlainTextRange(plain, localStartCol, localEndCol, visualBg)
				if rawIdx == m.cursorLine && isCursorInVisualRow(m.layout, visIdx, rawIdx, m.cursorCol) {
					localCursorCol := getLocalCursorCol(m.layout, visIdx, rawIdx, m.cursorCol)
					styled = stylePlainTextRangeWithCursor(plain, localStartCol, localEndCol, visualBg, localCursorCol, visualCursorBg, "")
				}
				sb.WriteString(decorateViewportLine(styled, m.vp.width, ""))
				continue
			}
			if rawIdx == m.cursorLine && isCursorInVisualRow(m.layout, visIdx, rawIdx, m.cursorCol) {
				plain := stripANSI(line)
				localCursorCol := getLocalCursorCol(m.layout, visIdx, rawIdx, m.cursorCol)
				sb.WriteString(decorateViewportLine(stylePlainTextRangeWithCursor(plain, 0, 0, "", localCursorCol, visualCursorBg, ""), m.vp.width, ""))
				continue
			}
		case visualModeLine:
			if start, end, ok := m.lineSelectionRange(); ok && rawIdx >= start && rawIdx <= end {
				if rawIdx == m.cursorLine && isCursorInVisualRow(m.layout, visIdx, rawIdx, m.cursorCol) {
					plain := stripANSI(line)
					localCursorCol := getLocalCursorCol(m.layout, visIdx, rawIdx, m.cursorCol)
					sb.WriteString(decorateViewportLine(stylePlainTextRangeWithCursor(plain, 0, lipgloss.Width(plain), visualBg, localCursorCol, visualCursorBg, ""), m.vp.width, visualBg))
					continue
				}
				sb.WriteString(decorateViewportLine(line, m.vp.width, visualBg))
				continue
			}
		}

		if rawIdx == m.cursorLine && isCursorInVisualRow(m.layout, visIdx, rawIdx, m.cursorCol) {
			plain := stripANSI(line)
			localCursorCol := getLocalCursorCol(m.layout, visIdx, rawIdx, m.cursorCol)
			sb.WriteString(decorateViewportLine(stylePlainTextRangeWithCursor(plain, 0, 0, "", localCursorCol, cursorCharBg, cursorLineBg), m.vp.width, cursorLineBg))
			continue
		}

		sb.WriteString(fitANSITextWidth(line, m.vp.width))
	}

	return sb.String()
}

// renderInputDock は入力欄部分（パディング行+入力行+パディング行）を構築する。
func (m *Model) renderInputDock() string {
	const inputBg = "\033[48;5;236m"
	tiView := strings.ReplaceAll(m.textInput.View(), "\033[0m", "\033[0m"+inputBg)
	inputLine := fillANSITextWidth(inputBg+" \033[38;5;46m"+inputPrompt+"\033[38;5;252m"+tiView+"\033[0m", m.width, inputBg)
	return m.padLineCache + "\n" + inputLine + "\n" + m.padLineCache
}

// renderStatusBar はステータスバー行を構築する。
func (m *Model) renderStatusBar() string {
	const hintColor = "\033[38;5;244m"
	statusLine := sanitizeSingleLineANSI(m.statusLine)

	var statusText string
	if m.navigationMode {
		statusText = " \033[48;5;33;38;5;255m NAV \033[0m " + statusLine
	} else if m.agent.IsProcessing() {
		statusText = " " + m.spinner.View() + " " + statusLine
	} else {
		statusText = " " + statusLine
	}

	if m.newOutput && !m.vp.atBottom() {
		statusText += "  \033[48;5;63;38;5;230m ↓ New output \033[0m"
	}

	if m.transientStatus != "" && time.Now().Before(m.transientStatusUntil) {
		statusText += "  \033[38;5;82m" + sanitizeSingleLineANSI(m.transientStatus) + "\033[0m"
	}

	hints := statusHintsNormal
	if m.hasActiveMouseSelection() || m.mouseDragging {
		hints = statusHintsMouseSel
	} else if m.navigationMode {
		if m.visualMode == visualModeChar {
			hints = statusHintsVisual
		} else if m.visualMode == visualModeLine {
			hints = statusHintsVisualLine
		} else if m.focusedBlock >= 0 {
			hints = statusHintsBlockFocus
		} else {
			hints = statusHintsNav
		}
	}
	statusBar := statusText
	for _, hint := range hints {
		padding := m.width - lipgloss.Width(statusText) - lipgloss.Width(hint)
		if padding >= 2 {
			statusBar = statusText + strings.Repeat(" ", padding) + hintColor + hint + "\033[0m"
			break
		}
	}
	return fitANSITextWidth(statusBar, m.width)
}

// rebuildChrome は入力欄+ステータスバーを再構築する。
// Update() 内で chromeDirty 時のみ呼ばれる（View() は値レシーバーなので書き込み不可）。
func (m *Model) rebuildChrome() {
	m.chromeCache = m.renderInputDock() + "\n" + m.renderStatusBar()
}

// View は bubbletea の View を実装する。
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if !m.ready {
		return "Initializing..."
	}
	return m.viewportView() + "\n" + m.chromeCache
}
