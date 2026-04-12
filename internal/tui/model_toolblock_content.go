package tui

// toggleToolBlock はツールブロックの折りたたみ/展開をトグルする。
func (m *Model) toggleToolBlock(blockIdx int) {
	if blockIdx < 0 || blockIdx >= len(m.toolBlocks) {
		return
	}

	m.toolBlocks[blockIdx].tool.Collapsed = !m.toolBlocks[blockIdx].tool.Collapsed
	m.replaceToolBlockLines(blockIdx, m.buildToolBlockLines(blockIdx))
	m.refreshToolBlockViewport()
}

func (m *Model) replaceToolBlockLines(blockIdx int, newLines []string) {
	block := &m.toolBlocks[blockIdx]
	oldCount := block.lineCount
	newCount := len(newLines)

	after := make([]string, len(m.rawLines[block.lineStart+oldCount:]))
	copy(after, m.rawLines[block.lineStart+oldCount:])
	m.rawLines = append(m.rawLines[:block.lineStart], newLines...)
	m.rawLines = append(m.rawLines, after...)

	block.lineCount = newCount
	delta := newCount - oldCount
	for i := blockIdx + 1; i < len(m.toolBlocks); i++ {
		m.toolBlocks[i].lineStart += delta
	}
}

func (m *Model) refreshToolBlockViewport() {
	m.rebuildLayout()
	if m.ready {
		m.vp.setLines(m.getVisualRowContents())
	}
	m.chromeDirty = true
}

// updateBlockIndicator はブロックのフォーカスインジケータを更新する。
func (m *Model) updateBlockIndicator(blockIdx int) {
	if blockIdx < 0 || blockIdx >= len(m.toolBlocks) {
		return
	}
	block := m.toolBlocks[blockIdx]
	if block.lineStart >= len(m.rawLines) {
		return
	}

	m.rawLines[block.lineStart] = m.toolBlockSummaryLine(blockIdx)
	m.refreshToolBlockViewport()
}
