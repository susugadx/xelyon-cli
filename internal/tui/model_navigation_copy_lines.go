package tui

import (
	"fmt"
	"strings"
)

func (m *Model) copyCursorLine() {
	if err := m.copyRawRangePlain(m.cursorLine, m.cursorLine); err != nil {
		m.setCopyError(err)
		return
	}
	m.setCopySuccess("Copied 1 line")
}

func (m *Model) copyDefaultSelectionTarget() {
	if msg, err := m.agent.CopyLastOutput(); err == nil {
		m.setCopySuccess(msg)
	} else {
		m.setCopyError(err)
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
