package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// appendToolResult はツール結果ブロックを追加する。
func (m *Model) appendToolResult(tool ToolResult) tea.Cmd {
	blockIdx := len(m.toolBlocks)
	lineStart := len(m.rawLines)

	block := toolBlockInfo{
		lineStart: lineStart,
		tool:      tool,
	}
	m.toolBlocks = append(m.toolBlocks, block)

	lines := m.buildToolBlockLines(blockIdx)
	m.toolBlocks[blockIdx].lineCount = len(lines)

	return m.appendContentLines(lines...)
}

// buildToolBlockLines はツールブロックの表示行を生成する。
func (m *Model) buildToolBlockLines(blockIdx int) []string {
	block := &m.toolBlocks[blockIdx]
	focused := m.focusedBlock == blockIdx

	indicator := " "
	if focused {
		indicator = "→"
	}

	prefix := "▶"
	if !block.tool.Collapsed {
		prefix = "▼"
	}

	summary := indicator + prefix + " " + block.tool.Summary

	if block.tool.Collapsed {
		return []string{summary}
	}

	lines := []string{summary}
	for _, line := range strings.Split(block.tool.Detail, "\n") {
		lines = append(lines, "  "+line)
	}
	return lines
}

// toggleToolBlock はツールブロックの折りたたみ/展開をトグルする。
func (m *Model) toggleToolBlock(blockIdx int) {
	if blockIdx < 0 || blockIdx >= len(m.toolBlocks) {
		return
	}
	block := &m.toolBlocks[blockIdx]
	block.tool.Collapsed = !block.tool.Collapsed

	newLines := m.buildToolBlockLines(blockIdx)
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

	m.rebuildRenderedLines()
	if m.ready {
		m.vp.setLines(m.renderedLines)
	}
}

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
}

// clearBlockFocus はブロックフォーカスを解除する。
func (m *Model) clearBlockFocus() {
	old := m.focusedBlock
	m.focusedBlock = -1
	m.updateBlockIndicator(old)
}

// moveBlockFocus はブロックフォーカスを移動する。
func (m *Model) moveBlockFocus(newIdx int) {
	newIdx = max(0, min(newIdx, len(m.toolBlocks)-1))
	if newIdx == m.focusedBlock {
		return
	}
	m.setBlockFocus(newIdx)
}

// updateBlockIndicator はブロックのフォーカスインジケータを更新する。
func (m *Model) updateBlockIndicator(blockIdx int) {
	if blockIdx < 0 || blockIdx >= len(m.toolBlocks) {
		return
	}
	block := &m.toolBlocks[blockIdx]
	focused := m.focusedBlock == blockIdx

	indicator := " "
	if focused {
		indicator = "→"
	}

	prefix := "▶"
	if !block.tool.Collapsed {
		prefix = "▼"
	}

	newFirstLine := indicator + prefix + " " + block.tool.Summary
	if block.lineStart < len(m.rawLines) {
		m.rawLines[block.lineStart] = newFirstLine
		if block.lineStart < len(m.renderedLines) {
			m.renderedLines[block.lineStart] = m.renderLine(newFirstLine)
		}
		if m.ready {
			m.vp.setLines(m.renderedLines)
		}
	}
}

// scrollToBlock はブロックの先頭行が表示されるようにスクロールする。
func (m *Model) scrollToBlock(blockIdx int) {
	if blockIdx < 0 || blockIdx >= len(m.toolBlocks) {
		return
	}
	block := &m.toolBlocks[blockIdx]
	target := max(0, block.lineStart-2)
	maxOffset := m.vp.maxYOffset()
	if target > maxOffset {
		target = maxOffset
	}
	m.vp.yOffset = target
}
