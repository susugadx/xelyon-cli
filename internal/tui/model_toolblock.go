package tui

import tea "github.com/charmbracelet/bubbletea"

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
