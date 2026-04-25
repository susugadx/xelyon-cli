package tui

import "github.com/susugadx/xelyon-cli/internal/tui/toolblock"

func (m Model) toolBlockSummaryLine(blockIdx int) string {
	block := m.toolBlocks[blockIdx]
	return toolblock.SummaryLine(block.tool.Summary, block.tool.Collapsed, m.focusedBlock == blockIdx)
}

// buildToolBlockLines はツールブロックの表示行を生成する。
func (m *Model) buildToolBlockLines(blockIdx int) []string {
	block := m.toolBlocks[blockIdx]
	return toolblock.Lines(block.tool.Summary, block.tool.Detail, block.tool.Collapsed, m.focusedBlock == blockIdx)
}
