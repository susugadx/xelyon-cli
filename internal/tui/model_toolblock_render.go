package tui

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
	"github.com/susugadx/xelyon-cli/internal/tui/toolblock"
	"github.com/susugadx/xelyon-cli/internal/uitoolview"
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
	summary := normalizeToolBlockSummary(tool.Summary)
	if summary == "" {
		summary = normalizeToolBlockSummary(tool.Name)
	}
	if summary == "" {
		summary = "tool"
	}

	if tool.Status != ToolStatusError && !tool.Error {
		return summary
	}
	errorSummary := normalizeToolBlockErrorSummary(summary)
	if errorSummary == "" {
		errorSummary = normalizeToolBlockSummary(tool.Name)
	}
	if errorSummary == "" {
		errorSummary = "tool failed"
	}
	return theme.Activity.ErrorTool + agentActivityErrorLabel(AgentErrorTool) + theme.Activity.Reset + " " + errorSummary
}

func normalizeToolBlockSummary(summary string) string {
	summary = termtext.SanitizeSingleLineANSI(strings.TrimSpace(summary))
	return uitoolview.StripToolDisplayIconPrefix(summary)
}

func normalizeToolBlockErrorSummary(summary string) string {
	summary = uitoolview.StripToolDisplayIconPrefix(summary)
	for {
		next := strings.TrimSpace(summary)
		lower := strings.ToLower(next)
		switch {
		case strings.HasPrefix(next, "✕ "):
			summary = strings.TrimSpace(strings.TrimPrefix(next, "✕ "))
		case strings.HasPrefix(lower, "error "):
			summary = strings.TrimSpace(next[len("error "):])
		case strings.HasPrefix(lower, "[tool error]"):
			summary = strings.TrimSpace(next[len("[tool error]"):])
		default:
			return next
		}
	}
}
