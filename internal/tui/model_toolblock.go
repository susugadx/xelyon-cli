package tui

import tea "github.com/charmbracelet/bubbletea"

// appendToolResult はツール結果ブロックを追加する。
func (m *Model) appendToolResult(tool ToolResult) tea.Cmd {
	if tool.Status == "" {
		if tool.Error {
			tool.Status = ToolStatusError
		} else {
			tool.Status = ToolStatusOK
		}
	}
	if tool.ID != "" {
		if idx := m.runningToolBlockIndexByID(tool.ID); idx >= 0 {
			return m.updateToolResult(idx, tool)
		}
	}

	blockIdx := len(m.toolBlocks)
	block := toolBlockInfo{
		tool: tool,
	}
	m.toolBlocks = append(m.toolBlocks, block)

	lines := m.buildToolBlockLines(blockIdx)
	m.chromeDirty = true

	return m.appendTrackedBlockLines(&m.toolBlocks[blockIdx].block, lines)
}

func (m Model) runningToolBlockIndexByID(id string) int {
	for i := len(m.toolBlocks) - 1; i >= 0; i-- {
		block := m.toolBlocks[i]
		if block.tool.ID == id && block.tool.Status == ToolStatusRunning {
			return i
		}
	}
	return -1
}

func (m *Model) updateToolResult(blockIdx int, tool ToolResult) tea.Cmd {
	if blockIdx < 0 || blockIdx >= len(m.toolBlocks) {
		return nil
	}
	if !tool.Error && tool.Status != ToolStatusError {
		tool.Collapsed = m.toolBlocks[blockIdx].tool.Collapsed
	}
	m.toolBlocks[blockIdx].tool = tool
	m.updateTrackedBlockLinesFollowing(&m.toolBlocks[blockIdx].block, m.buildToolBlockLines(blockIdx))
	return nil
}
