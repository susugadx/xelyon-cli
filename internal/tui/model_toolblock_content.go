package tui

// toggleToolBlock はツールブロックの折りたたみ/展開をトグルする。
func (m *Model) toggleToolBlock(blockIdx int) {
	if blockIdx < 0 || blockIdx >= len(m.toolBlocks) {
		return
	}

	m.toolBlocks[blockIdx].tool.Collapsed = !m.toolBlocks[blockIdx].tool.Collapsed
	m.replaceTrackedBlockLines(&m.toolBlocks[blockIdx].block, m.buildToolBlockLines(blockIdx))
	m.refreshTranscriptViewport()
}

// updateBlockIndicator はブロックのフォーカスインジケータを更新する。
func (m *Model) updateBlockIndicator(blockIdx int) {
	if blockIdx < 0 || blockIdx >= len(m.toolBlocks) {
		return
	}
	toolBlock := m.toolBlocks[blockIdx]
	if !m.replaceTrackedBlockFirstLine(toolBlock.block, m.toolBlockSummaryLine(blockIdx)) {
		return
	}

	m.refreshTranscriptViewport()
}
