package tui

import (
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
	"github.com/susugadx/xelyon-cli/internal/tui/toolblock"
)

func (m Model) toolBlockSummaryLine(blockIdx int) string {
	block := m.toolBlocks[blockIdx]
	return toolblock.SummaryLine(toolBlockDisplaySummary(block.tool), block.tool.Collapsed, m.focusedBlock == blockIdx)
}

// buildToolBlockLines はツールブロックの表示行を生成する。
func (m *Model) buildToolBlockLines(blockIdx int) []string {
	block := m.toolBlocks[blockIdx]
	return toolblock.Lines(toolBlockDisplaySummary(block.tool), block.tool.Detail, block.tool.Collapsed, m.focusedBlock == blockIdx)
}

func toolBlockDisplaySummary(tool ToolResult) string {
	if tool.Status != ToolStatusError && !tool.Error {
		return tool.Summary
	}
	return theme.Activity.ErrorTool + agentActivityErrorLabel(AgentErrorTool) + theme.Activity.Reset + " " + tool.Summary
}
