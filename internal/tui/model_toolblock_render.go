package tui

import "strings"

func (m Model) toolBlockSummaryLine(blockIdx int) string {
	block := m.toolBlocks[blockIdx]

	indicator := " "
	if m.focusedBlock == blockIdx {
		indicator = "→"
	}

	prefix := "▶"
	if !block.tool.Collapsed {
		prefix = "▼"
	}

	return indicator + prefix + " " + block.tool.Summary
}

// buildToolBlockLines はツールブロックの表示行を生成する。
func (m *Model) buildToolBlockLines(blockIdx int) []string {
	summary := m.toolBlockSummaryLine(blockIdx)
	if m.toolBlocks[blockIdx].tool.Collapsed {
		return []string{summary}
	}

	lines := []string{summary}
	for _, line := range strings.Split(m.toolBlocks[blockIdx].tool.Detail, "\n") {
		lines = append(lines, "  "+line)
	}
	return lines
}
