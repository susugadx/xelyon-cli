package tui

import "strings"

func (m *Model) copyVisualSelection() {
	switch m.visualMode {
	case visualModeChar:
		text, lines := m.copyCharVisualSelectionText()
		if err := m.agent.CopyText(text); err != nil {
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
