package tui

// setBlockFocus はブロックフォーカスを設定する。
func (m *Model) setBlockFocus(blockIdx int) {
	if blockIdx < 0 || blockIdx >= len(m.toolBlocks) {
		return
	}

	m.clearVisualSelection()
	old := m.focusedBlock
	m.focusedBlock = blockIdx
	m.cursorLine = m.toolBlocks[blockIdx].lineStart
	m.updateBlockIndicator(old)
	m.updateBlockIndicator(m.focusedBlock)
	m.scrollToBlock(m.focusedBlock)
	m.chromeDirty = true
}

// clearBlockFocus はブロックフォーカスを解除する。
func (m *Model) clearBlockFocus() {
	old := m.focusedBlock
	m.focusedBlock = -1
	m.updateBlockIndicator(old)
	m.chromeDirty = true
}

// moveBlockFocus はブロックフォーカスを移動する。
func (m *Model) moveBlockFocus(newIdx int) {
	newIdx = max(0, min(newIdx, len(m.toolBlocks)-1))
	if newIdx == m.focusedBlock {
		return
	}

	m.setBlockFocus(newIdx)
}

// scrollToBlock はブロックの先頭行が表示されるようにスクロールする。
func (m *Model) scrollToBlock(blockIdx int) {
	if blockIdx < 0 || blockIdx >= len(m.toolBlocks) {
		return
	}

	block := m.toolBlocks[blockIdx]
	target := 0
	if m.layout != nil && block.lineStart < len(m.layout.LineToRowMap) {
		target = max(0, m.layout.LineToRowMap[block.lineStart]-2)
	}
	maxOffset := m.vp.maxYOffset()
	if target > maxOffset {
		target = maxOffset
	}
	m.vp.yOffset = target
	if m.vp.atBottom() && m.newOutput {
		m.newOutput = false
		m.chromeDirty = true
	}
}
