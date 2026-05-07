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
	lineStart := len(m.rawLines)

	block := toolBlockInfo{
		lineStart: lineStart,
		tool:      tool,
	}
	m.toolBlocks = append(m.toolBlocks, block)

	lines := m.buildToolBlockLines(blockIdx)
	m.toolBlocks[blockIdx].lineCount = len(lines)
	m.chromeDirty = true

	return m.appendContentLines(lines...)
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
	follow := m.captureViewportFollowState()
	m.toolBlocks[blockIdx].tool = tool
	m.replaceToolBlockLines(blockIdx, m.buildToolBlockLines(blockIdx))
	m.refreshToolBlockViewportFollowing(follow)
	return nil
}
