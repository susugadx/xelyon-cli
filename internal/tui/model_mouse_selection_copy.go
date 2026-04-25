package tui

import "github.com/susugadx/xelyon-cli/internal/tui/selection"

// mouseSelectionText はマウス選択範囲のテキストを返す。
// コピー実行時にのみ呼ばれ、ドラッグ中は遅延評価する。
func (m Model) mouseSelectionText() (string, int) {
	r, ok := selection.Normalize(m.mouseSelAnchor.line, m.mouseSelAnchor.col, m.mouseSelEnd.line, m.mouseSelEnd.col)
	if !ok {
		return "", 0
	}
	return selection.ANSIPlainText(m.rawLines, r)
}

// copyMouseSelection はマウス選択範囲をクリップボードにコピーする。
func (m *Model) copyMouseSelection() {
	text, lines := m.mouseSelectionText()
	if text == "" {
		return
	}
	if err := m.clipboard.CopyText(text); err != nil {
		m.setTransientStatus("Copy failed: " + err.Error())
		return
	}
	m.clearMouseSelection()
	m.setTransientStatus("✅ Copied " + lineLabel(lines))
	m.chromeDirty = true
}
