package tui

import (
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rivo/uniseg"
)

func normalizeRawLine(line string) string {
	return strings.TrimSuffix(line, "\r")
}

func isANSITerminator(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

type streamCell struct {
	text  string
	style string
	span  int
}

func applyANSICode(current, code string) string {
	if !strings.HasSuffix(code, "m") {
		return current
	}
	if code == "\033[0m" || code == "\033[m" {
		return ""
	}
	return current + code
}

func parseStreamCells(s string) []streamCell {
	cells := make([]streamCell, 0, len(s))
	inEscape := false
	var token strings.Builder
	currentStyle := ""

	appendCluster := func(text string, width int) {
		if width <= 0 {
			return
		}
		cells = append(cells, streamCell{text: text, style: currentStyle, span: width})
		for i := 1; i < width; i++ {
			cells = append(cells, streamCell{style: currentStyle, span: 0})
		}
	}

	flushSegment := func(segment string) {
		gr := uniseg.NewGraphemes(segment)
		for gr.Next() {
			cluster := gr.Str()
			if cluster == "\t" {
				appendCluster("\t", visualTabWidth)
				continue
			}
			appendCluster(cluster, plainTextDisplayWidth(cluster))
		}
	}

	segmentStart := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			size = 1
		}

		if inEscape {
			token.WriteRune(r)
			if isANSITerminator(r) {
				inEscape = false
				currentStyle = applyANSICode(currentStyle, token.String())
				token.Reset()
				segmentStart = i + size
			}
			i += size
			continue
		}

		if r == '\033' {
			if segmentStart < i {
				flushSegment(s[segmentStart:i])
			}
			inEscape = true
			token.WriteRune(r)
			i += size
			continue
		}

		i += size
	}

	if !inEscape && segmentStart < len(s) {
		flushSegment(s[segmentStart:])
	}

	return cells
}

func rebuildStreamLine(cells []streamCell) string {
	var b strings.Builder
	currentStyle := ""
	for _, cell := range cells {
		if cell.span == 0 {
			continue
		}
		if cell.style != currentStyle {
			if currentStyle != "" {
				b.WriteString("\033[0m")
			}
			if cell.style != "" {
				b.WriteString(cell.style)
			}
			currentStyle = cell.style
		}
		b.WriteString(cell.text)
	}
	if currentStyle != "" {
		b.WriteString("\033[0m")
	}
	return b.String()
}

func streamCellStartIndex(cells []streamCell, idx int) int {
	if idx < 0 {
		return -1
	}
	if idx >= len(cells) {
		idx = len(cells) - 1
	}
	for idx >= 0 {
		if cells[idx].span > 0 {
			return idx
		}
		idx--
	}
	return -1
}

func applyClusterToStreamCells(cells []streamCell, col int, cluster string, width int, style string) []streamCell {
	if col < 0 {
		col = 0
	}
	if width <= 0 {
		start := streamCellStartIndex(cells, col-1)
		if start >= 0 {
			cells[start].text += cluster
			if style != "" && cells[start].style == "" {
				cells[start].style = style
			}
			return cells
		}
		if style != "" {
			return append(cells, streamCell{text: cluster, style: style, span: 1})
		}
		return append(cells, streamCell{text: cluster, span: 1})
	}
	for len(cells) < col {
		cells = append(cells, streamCell{text: " ", span: 1})
	}
	for len(cells) < col+width {
		cells = append(cells, streamCell{text: " ", span: 1})
	}
	for i := 0; i < width; i++ {
		target := col + i
		if target >= len(cells) {
			break
		}
		start := target
		if cells[target].span == 0 {
			for start >= 0 {
				if cells[start].span > 0 && start+cells[start].span > target {
					break
				}
				start--
			}
			if start < 0 {
				start = target
			}
		}
		span := 1
		if cells[start].span > 0 {
			span = cells[start].span
		}
		for j := start; j < start+span && j < len(cells); j++ {
			cells[j] = streamCell{text: " ", style: cells[j].style, span: 1}
		}
	}
	cells[col] = streamCell{text: cluster, style: style, span: width}
	for i := col + 1; i < col+width; i++ {
		cells[i] = streamCell{style: style, span: 0}
	}
	return cells
}

func mergeStreamFragment(currentLine, fragment string, cursorCol int, currentANSI, pendingPartialANSI string) (string, int, string, string) {
	if fragment == "" {
		if cursorCol < 0 {
			cursorCol = 0
		}
		return currentLine, cursorCol, currentANSI, pendingPartialANSI
	}

	if cursorCol < 0 {
		cursorCol = 0
	}

	fragment = pendingPartialANSI + fragment
	cells := parseStreamCells(currentLine)
	inEscape := false
	var partialANSI strings.Builder
	segmentStart := 0

	flushSegment := func(segment string) {
		if segment == "" {
			return
		}
		gr := uniseg.NewGraphemes(segment)
		for gr.Next() {
			cluster := gr.Str()
			width := plainTextDisplayWidth(cluster)
			cells = applyClusterToStreamCells(cells, cursorCol, cluster, width, currentANSI)
			cursorCol += width
		}
	}

	for i := 0; i < len(fragment); {
		r, size := utf8.DecodeRuneInString(fragment[i:])
		if r == utf8.RuneError && size == 1 {
			size = 1
		}

		if inEscape {
			partialANSI.WriteRune(r)
			if isANSITerminator(r) {
				inEscape = false
				currentANSI = applyANSICode(currentANSI, partialANSI.String())
				partialANSI.Reset()
				segmentStart = i + size
			}
			i += size
			continue
		}

		if r == '\033' {
			if segmentStart < i {
				flushSegment(fragment[segmentStart:i])
			}
			inEscape = true
			partialANSI.WriteRune(r)
			i += size
			continue
		}

		if r == '\r' {
			if segmentStart < i {
				flushSegment(fragment[segmentStart:i])
			}
			cursorCol = 0
			segmentStart = i + size
			i += size
			continue
		}
		i += size
	}

	line := rebuildStreamLine(cells)
	if inEscape {
		return line, cursorCol, currentANSI, partialANSI.String()
	}
	if segmentStart < len(fragment) {
		flushSegment(fragment[segmentStart:])
		line = rebuildStreamLine(cells)
	}

	return line, cursorCol, currentANSI, ""
}

func (m *Model) appendStreamLines(parts []string) tea.Cmd {
	lines := make([]string, 0, len(parts))
	firstLine, cursorCol, activeANSI, pendingANSI := mergeStreamFragment("", parts[0], 0, m.streamActiveANSI, m.streamPendingANSI)
	lines = append(lines, firstLine)
	m.streamCursorCol = cursorCol
	m.streamActiveANSI = activeANSI
	m.streamPendingANSI = pendingANSI
	for _, part := range parts[1:] {
		line, nextCursor, nextActiveANSI, nextPendingANSI := mergeStreamFragment("", part, 0, m.streamActiveANSI, "")
		lines = append(lines, line)
		m.streamCursorCol = nextCursor
		m.streamActiveANSI = nextActiveANSI
		m.streamPendingANSI = nextPendingANSI
	}
	return m.appendContentLines(lines...)
}

// appendMessage は会話ログにメッセージを追加する。
func (m *Model) appendMessage(msg ChatMessage) tea.Cmd {
	m.messages = append(m.messages, msg)

	switch msg.Role {
	case "user":
		return m.appendContentLines("", "> "+msg.Content, "")
	default:
		return m.appendContentLines(strings.Split(msg.Content, "\n")...)
	}
}

// appendStreamText はストリーミングテキストを追加する。
func (m *Model) appendStreamText(text string) tea.Cmd {
	if text == "" {
		return nil
	}
	parts := strings.Split(text, "\n")

	if !m.streamingActive {
		m.streamingActive = true
		return m.appendStreamLines(parts)
	}

	if len(m.rawLines) == 0 {
		return m.appendStreamLines(parts)
	}

	last := len(m.rawLines) - 1
	m.rawLines[last], m.streamCursorCol, m.streamActiveANSI, m.streamPendingANSI = mergeStreamFragment(m.rawLines[last], parts[0], m.streamCursorCol, m.streamActiveANSI, m.streamPendingANSI)
	m.rebuildLayout()
	if len(parts) > 1 {
		lines := make([]string, 0, len(parts)-1)
		for _, part := range parts[1:] {
			line, nextCursor, nextActiveANSI, pendingANSI := mergeStreamFragment("", part, 0, m.streamActiveANSI, "")
			lines = append(lines, line)
			m.streamCursorCol = nextCursor
			m.streamActiveANSI = nextActiveANSI
			m.streamPendingANSI = pendingANSI
		}
		_ = m.appendContentLines(lines...)
	}
	m.clampCursorLine()
	if m.ready {
		atBottom := m.vp.atBottom()
		m.vp.setLines(m.getVisualRowContents())
		if atBottom {
			m.vp.gotoBottom()
			m.newOutput = false
		} else {
			m.newOutput = true
		}
	}
	return nil
}

// appendSystemInfo はシステム情報メッセージを追加する。
func (m *Model) appendSystemInfo(text string) tea.Cmd {
	return m.appendMessage(ChatMessage{
		Role:      "system_info",
		Content:   text,
		Timestamp: time.Now(),
	})
}

// appendContentLines は生ログと描画済みログの両方に新しい行を追加する。
func (m *Model) appendContentLines(lines ...string) tea.Cmd {
	if len(lines) == 0 {
		return nil
	}
	normalized := make([]string, len(lines))
	for i, line := range lines {
		normalized[i] = normalizeRawLine(line)
	}
	m.rawLines = append(m.rawLines, normalized...)
	m.rebuildLayout()
	m.clampCursorLine()
	if m.ready {
		atBottom := m.vp.atBottom()
		m.vp.setLines(m.getVisualRowContents())
		if atBottom {
			m.vp.gotoBottom()
			m.newOutput = false
		} else {
			m.newOutput = true
		}
	}
	return nil
}

func (m *Model) clampCursorToViewport() {
	if len(m.rawLines) == 0 || m.vp.height <= 0 {
		m.cursorLine = 0
		m.cursorCol = 0
		return
	}
	top := 0
	if m.layout != nil && m.vp.yOffset < len(m.layout.Rows) {
		top = m.layout.Rows[m.vp.yOffset].RawLineIdx
	}
	bottomIdx := m.vp.yOffset + m.vp.height - 1
	if m.layout != nil && bottomIdx >= len(m.layout.Rows) {
		bottomIdx = len(m.layout.Rows) - 1
	}
	bottom := len(m.rawLines) - 1
	if m.layout != nil && bottomIdx >= 0 && bottomIdx < len(m.layout.Rows) {
		bottom = m.layout.Rows[bottomIdx].RawLineIdx
	}
	if m.cursorLine < top {
		m.cursorLine = top
	}
	if m.cursorLine > bottom {
		m.cursorLine = bottom
	}
	m.clampCursorCol()
}
