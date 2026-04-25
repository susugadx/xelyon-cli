package tui

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

// renderViewportWithMouseSelection はマウス選択オーバーレイ付きの viewport を描画する。
// 可視行のみに選択背景を適用し、スクロール性能を維持する。
func (m Model) renderViewportWithMouseSelection() string {
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

		if startCol, endCol, ok := m.mouseSelectionColumnsForLine(rawIdx); ok {
			plain := termtext.StripANSI(line)
			localStart, localEnd := m.mouseSelectionColumnsForVisualRow(visIdx, rawIdx, startCol, endCol)
			if localStart < localEnd {
				styled := termtext.StylePlainTextRange(plain, localStart, localEnd, theme.Viewport.MouseSelectionBg)
				sb.WriteString(decorateViewportLine(styled, m.vp.width, ""))
				continue
			}
		}

		sb.WriteString(fitANSITextWidth(line, m.vp.width))
	}

	return sb.String()
}
