package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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
	m.syncViewportContent()
	return nil
}
