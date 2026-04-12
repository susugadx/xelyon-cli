package tui

import "strings"

// mouseSelectionText はマウス選択範囲のテキストを返す。
// コピー実行時にのみ呼ばれ、ドラッグ中は遅延評価する。
func (m Model) mouseSelectionText() (string, int) {
	start, end, ok := m.normalizedMouseSelection()
	if !ok || len(m.rawLines) == 0 {
		return "", 0
	}

	var result strings.Builder
	lineCount := 0
	for i := start.line; i <= end.line && i < len(m.rawLines); i++ {
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

		if lineCount > 0 {
			result.WriteByte('\n')
		}
		result.WriteString(string(runes[from:to]))
		lineCount++
	}
	return result.String(), lineCount
}

// copyMouseSelection はマウス選択範囲をクリップボードにコピーする。
func (m *Model) copyMouseSelection() {
	text, lines := m.mouseSelectionText()
	if text == "" {
		return
	}
	if err := m.agent.CopyText(text); err != nil {
		m.setTransientStatus("Copy failed: " + err.Error())
		return
	}
	m.clearMouseSelection()
	m.setTransientStatus("✅ Copied " + lineLabel(lines))
	m.chromeDirty = true
}
