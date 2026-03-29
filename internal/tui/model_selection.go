package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// mouseAutoScrollMsg はマウスドラッグ中の自動スクロールティック。
type mouseAutoScrollMsg struct{}

const (
	autoScrollEdgeZone = 3                     // viewport 端からの行数
	autoScrollInterval = 50 * time.Millisecond // 自動スクロールの間隔
	autoScrollMaxSpeed = 5                     // 最大スクロール速度（行/tick）
	mouseSelBg         = "\033[48;5;240m"      // マウス選択背景（visual mode と統一）
)

// hasActiveMouseSelection はマウス選択がアクティブかどうかを返す。
func (m Model) hasActiveMouseSelection() bool {
	if m.mouseSelAnchor.line < 0 {
		return false
	}
	return m.mouseSelAnchor.line != m.mouseSelEnd.line ||
		m.mouseSelAnchor.col != m.mouseSelEnd.col
}

// clearMouseSelection はマウス選択をクリアする。
func (m *Model) clearMouseSelection() {
	m.mouseSelAnchor = visualPosition{line: -1, col: -1}
	m.mouseSelEnd = visualPosition{line: -1, col: -1}
	m.mouseDragging = false
	m.mouseAutoScrolling = false
}

// normalizedMouseSelection は正規化されたマウス選択範囲を返す（start <= end）。
func (m Model) normalizedMouseSelection() (start, end visualPosition, ok bool) {
	if !m.hasActiveMouseSelection() {
		return visualPosition{}, visualPosition{}, false
	}
	start = m.mouseSelAnchor
	end = m.mouseSelEnd
	if start.line > end.line || (start.line == end.line && start.col > end.col) {
		start, end = end, start
	}
	return start, end, true
}

// mouseSelectionColumnsForLine は指定行のマウス選択列範囲を返す。
// startCol は包含、endCol は排他。選択に含まれない行は ok=false を返す。
func (m Model) mouseSelectionColumnsForLine(line int) (startCol, endCol int, ok bool) {
	start, end, ok := m.normalizedMouseSelection()
	if !ok || line < start.line || line > end.line || line >= len(m.rawLines) {
		return 0, 0, false
	}
	switch {
	case start.line == end.line:
		return start.col, end.col + 1, true
	case line == start.line:
		return start.col, 9999, true
	case line == end.line:
		return 0, end.col + 1, true
	default:
		return 0, 9999, true
	}
}

// mouseSelectionColumnsForVisualRow は rawLine レベルの選択列をビジュアル行ローカル列に変換する。
// ラップされた行の prefix を考慮し、可視行に対する正しい列範囲を返す。
func (m Model) mouseSelectionColumnsForVisualRow(visIdx, rawIdx, startCol, endCol int) (int, int) {
	if m.layout == nil || visIdx >= len(m.layout.Rows) {
		return startCol, endCol
	}

	startVisRow, startVisCol := m.layout.GetVisualCursor(rawIdx, startCol)
	endVisRow, endVisCol := m.layout.GetVisualCursor(rawIdx, endCol)

	// 排他的 endCol がラップ行の prefix 位置に来た場合、前行の末尾に補正
	if endVisRow > startVisRow && endVisRow >= 0 && endVisRow < len(m.layout.Rows) {
		if endVisCol == m.layout.Rows[endVisRow].PrefixWidth {
			endVisRow--
			endVisCol = m.vp.width
		}
	}

	if visIdx < startVisRow || visIdx > endVisRow {
		return 0, 0
	}

	localStart := 0
	localEnd := m.vp.width
	if visIdx == startVisRow {
		localStart = startVisCol
	}
	if visIdx == endVisRow {
		localEnd = endVisCol
	}
	return localStart, localEnd
}

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

// screenToRawPosition はスクリーン座標を rawLine/rawCol に変換する。
// screenY は viewport 内の相対 Y 座標。
func (m Model) screenToRawPosition(screenX, screenY int) (rawLine, rawCol int, ok bool) {
	if m.layout == nil || len(m.layout.Rows) == 0 {
		return 0, 0, false
	}

	visRowIdx := m.vp.yOffset + screenY
	if visRowIdx < 0 {
		visRowIdx = 0
	}
	if visRowIdx >= len(m.layout.Rows) {
		visRowIdx = len(m.layout.Rows) - 1
	}

	row := m.layout.Rows[visRowIdx]
	rawLine = row.RawLineIdx

	baseCol := m.layout.GetRawColumnForVisualRow(visRowIdx)
	localX := screenX
	if localX < 0 {
		localX = 0
	}
	if localX < row.PrefixWidth {
		localX = row.PrefixWidth
	}
	rawCol = baseCol + (localX - row.PrefixWidth)

	maxCol := m.maxCursorColForLine(rawLine)
	if rawCol > maxCol {
		rawCol = maxCol
	}
	if rawCol < 0 {
		rawCol = 0
	}
	return rawLine, rawCol, true
}

// handleMouseSelection はマウスイベントの選択処理を行う。
// transcript viewport 領域でのみ選択を扱い、input dock 上では無視する。
func (m *Model) handleMouseSelection(msg tea.MouseMsg) tea.Cmd {
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return nil
		}
		if msg.Y >= m.vp.height {
			return nil // Input dock 上では選択開始しない
		}
		rawLine, rawCol, ok := m.screenToRawPosition(msg.X, msg.Y)
		if !ok {
			return nil
		}
		// 既存の keyboard visual selection をクリア
		if m.visualMode != visualModeOff {
			m.clearVisualSelection()
		}
		m.resetNavPending()
		m.mouseSelAnchor = visualPosition{line: rawLine, col: rawCol}
		m.mouseSelEnd = m.mouseSelAnchor
		m.mouseDragging = true
		m.mouseAutoScrolling = false
		m.mouseLastScreenX = msg.X
		m.mouseLastScreenY = msg.Y
		m.chromeDirty = true
		return nil

	case tea.MouseActionMotion:
		if !m.mouseDragging {
			return nil
		}
		m.mouseLastScreenX = msg.X
		m.mouseLastScreenY = msg.Y

		clampedY := msg.Y
		if clampedY < 0 {
			clampedY = 0
		}
		if clampedY >= m.vp.height {
			clampedY = m.vp.height - 1
		}
		if rawLine, rawCol, ok := m.screenToRawPosition(msg.X, clampedY); ok {
			m.mouseSelEnd = visualPosition{line: rawLine, col: rawCol}
		}
		m.chromeDirty = true

		// Edge auto-scroll: viewport 端に近づいたら自動スクロール開始
		atEdge := msg.Y < autoScrollEdgeZone || msg.Y >= m.vp.height-autoScrollEdgeZone
		if atEdge && !m.mouseAutoScrolling {
			m.mouseAutoScrolling = true
			return mouseAutoScrollTick()
		}
		if !atEdge {
			m.mouseAutoScrolling = false
		}
		return nil

	case tea.MouseActionRelease:
		if !m.mouseDragging {
			return nil
		}
		m.mouseDragging = false
		m.mouseAutoScrolling = false

		clampedY := msg.Y
		if clampedY < 0 {
			clampedY = 0
		}
		if clampedY >= m.vp.height {
			clampedY = m.vp.height - 1
		}
		if rawLine, rawCol, ok := m.screenToRawPosition(msg.X, clampedY); ok {
			m.mouseSelEnd = visualPosition{line: rawLine, col: rawCol}
		}

		// クリック（ドラッグなし）は選択クリア
		if m.mouseSelAnchor == m.mouseSelEnd {
			m.clearMouseSelection()
		}
		m.chromeDirty = true
		return nil
	}
	return nil
}

// autoScrollAmount は現在のマウス位置からのスクロール量を返す。
// 正の値は下方向、負の値は上方向。端への近さに応じて加速する。
func (m Model) autoScrollAmount() int {
	y := m.mouseLastScreenY
	h := m.vp.height
	if h <= 0 {
		return 0
	}

	if y < autoScrollEdgeZone {
		speed := autoScrollEdgeZone - y
		if speed > autoScrollMaxSpeed {
			speed = autoScrollMaxSpeed
		}
		return -speed
	}
	if y >= h-autoScrollEdgeZone {
		speed := y - (h - autoScrollEdgeZone) + 1
		if speed > autoScrollMaxSpeed {
			speed = autoScrollMaxSpeed
		}
		return speed
	}
	return 0
}

// handleAutoScroll はドラッグ中の自動スクロールを処理する。
// viewport をスクロールし、選択端を更新して次のティックを発行する。
func (m *Model) handleAutoScroll() tea.Cmd {
	if !m.mouseDragging {
		m.mouseAutoScrolling = false
		return nil
	}

	amount := m.autoScrollAmount()
	if amount == 0 {
		m.mouseAutoScrolling = false
		return nil
	}

	if amount < 0 {
		m.vp.scrollUp(-amount)
	} else {
		m.vp.scrollDown(amount)
	}

	// スクロール後の位置で selection end を更新
	clampedY := m.mouseLastScreenY
	if clampedY < 0 {
		clampedY = 0
	}
	if clampedY >= m.vp.height {
		clampedY = m.vp.height - 1
	}
	if rawLine, rawCol, ok := m.screenToRawPosition(m.mouseLastScreenX, clampedY); ok {
		m.mouseSelEnd = visualPosition{line: rawLine, col: rawCol}
	}

	m.afterViewportScroll()
	return mouseAutoScrollTick()
}

func mouseAutoScrollTick() tea.Cmd {
	return tea.Tick(autoScrollInterval, func(t time.Time) tea.Msg {
		return mouseAutoScrollMsg{}
	})
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
			plain := stripANSI(line)
			localStart, localEnd := m.mouseSelectionColumnsForVisualRow(visIdx, rawIdx, startCol, endCol)
			if localStart < localEnd {
				styled := stylePlainTextRange(plain, localStart, localEnd, mouseSelBg)
				sb.WriteString(decorateViewportLine(styled, m.vp.width, ""))
				continue
			}
		}

		sb.WriteString(fitANSITextWidth(line, m.vp.width))
	}

	return sb.String()
}
