package tui

import "github.com/susugadx/xelyon-cli/internal/tui/selection"

func (m *Model) copyVisualSelection() {
	switch m.visualMode {
	case visualModeChar:
		text, lines := m.copyCharVisualSelectionText()
		if err := m.clipboard.CopyText(text); err != nil {
			m.setCopyError(err)
			return
		}
		m.clearVisualSelection()
		m.setCopySuccess("Copied " + lineLabel(lines))
	case visualModeLine:
		start, end, ok := m.lineSelectionRange()
		if !ok {
			return
		}
		if err := m.copyRawRangePlain(start, end); err != nil {
			m.setCopyError(err)
			return
		}
		m.clearVisualSelection()
		m.setCopySuccess("Copied " + lineLabel(end-start+1))
	}
}

func (m Model) copyCharVisualSelectionText() (string, int) {
	start, end, ok := m.normalizedCharSelection()
	if !ok {
		return "", 0
	}
	return selection.ANSIPlainText(m.rawLines, selection.Range{
		StartLine: start.line,
		StartCol:  start.col,
		EndLine:   end.line,
		EndCol:    end.col,
	})
}
