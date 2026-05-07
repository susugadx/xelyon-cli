package tui

import tea "github.com/charmbracelet/bubbletea"

// appendTrackedBlockLines は rawLines 末尾に更新可能な block を追加する。
func (m *Model) appendTrackedBlockLines(block *trackedBlock, lines []string) tea.Cmd {
	block.lineStart = len(m.rawLines)
	block.lineCount = len(lines)
	return m.appendContentLines(lines...)
}

// updateTrackedBlockLinesFollowing は viewport の bottom follow 状態を保って block 行を更新する。
func (m *Model) updateTrackedBlockLinesFollowing(block *trackedBlock, newLines []string) {
	follow := m.captureViewportFollowState()
	m.replaceTrackedBlockLines(block, newLines)
	m.refreshTranscriptViewportFollowing(follow)
}

// replaceTrackedBlockLines は rawLines 上の block 範囲を差し替え、後続 block の開始位置を補正する。
func (m *Model) replaceTrackedBlockLines(block *trackedBlock, newLines []string) {
	oldCount := block.lineCount
	newCount := len(newLines)

	after := make([]string, len(m.rawLines[block.lineStart+oldCount:]))
	copy(after, m.rawLines[block.lineStart+oldCount:])
	m.rawLines = append(m.rawLines[:block.lineStart], newLines...)
	m.rawLines = append(m.rawLines, after...)

	block.lineCount = newCount
	m.shiftTrackedBlocksAfter(block.lineStart, newCount-oldCount)
}

func (m *Model) replaceTrackedBlockFirstLine(block trackedBlock, line string) bool {
	if block.lineStart >= len(m.rawLines) {
		return false
	}
	m.rawLines[block.lineStart] = line
	return true
}

func (m *Model) shiftTrackedBlocksAfter(lineStart int, delta int) {
	if delta == 0 {
		return
	}
	for i := range m.toolBlocks {
		if m.toolBlocks[i].block.lineStart > lineStart {
			m.toolBlocks[i].block.lineStart += delta
		}
	}
}

func (m *Model) refreshTranscriptViewport() {
	m.rebuildLayout()
	if m.ready {
		m.vp.setLines(m.getVisualRowContents())
	}
	m.chromeDirty = true
}

func (m *Model) refreshTranscriptViewportFollowing(follow viewportFollowState) {
	m.rebuildLayout()
	m.syncViewportContentFrom(follow)
	m.chromeDirty = true
}
