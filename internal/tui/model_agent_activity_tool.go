package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/uitoolview"
)

func (m *Model) upsertAgentActivityTool(tool ToolResult) tea.Cmd {
	if !m.hasActiveAgentActivity() {
		return m.appendToolResult(tool)
	}

	tool = normalizeActivityTool(tool)
	if idx := m.agentActivityToolIndex(tool.ID); idx >= 0 {
		tool = inheritActivityToolTiming(m.agentActivity.tools[idx].tool, tool)
		m.agentActivity.tools[idx].tool = tool
	} else {
		m.agentActivity.tools = append(m.agentActivity.tools, agentActivityTool{tool: tool})
	}

	if (tool.Status == ToolStatusError || tool.Error) && !tool.NonBlockingError {
		m.agentActivity.status = agentActivityStatusBlocked
		m.agentActivity.errorKind = AgentErrorTool
		if m.agentActivity.errorText == "" {
			m.agentActivity.errorText = agentActivityToolErrorText(tool)
		}
	}

	m.updateTrackedBlockLinesFollowing(&m.agentActivity.block, m.buildAgentActivityLines(time.Now()))
	m.chromeDirty = true
	return nil
}

func normalizeActivityTool(tool ToolResult) ToolResult {
	tool.Name = termtext.SanitizeSingleLineANSI(strings.TrimSpace(tool.Name))
	if tool.Name == "" {
		tool.Name = "tool"
	}
	tool.Target = termtext.SanitizeSingleLineANSI(strings.TrimSpace(tool.Target))
	if tool.Status == "" {
		if tool.Error {
			tool.Status = ToolStatusError
		} else {
			tool.Status = ToolStatusOK
		}
	}
	if tool.Status == ToolStatusError {
		tool.Error = true
	}
	return tool
}

func inheritActivityToolTiming(previous ToolResult, next ToolResult) ToolResult {
	if next.StartedAt.IsZero() {
		next.StartedAt = previous.StartedAt
	}
	if next.Duration == 0 && next.Status != ToolStatusRunning && !next.StartedAt.IsZero() {
		next.Duration = time.Since(next.StartedAt)
	}
	return next
}

func (m Model) agentActivityToolIndex(id string) int {
	if id == "" {
		return -1
	}
	for i := range m.agentActivity.tools {
		if m.agentActivity.tools[i].tool.ID == id {
			return i
		}
	}
	return -1
}

func agentActivityToolErrorText(tool ToolResult) string {
	if detail := termtext.SanitizeSingleLineANSI(strings.TrimSpace(firstLine(tool.Detail))); detail != "" {
		return detail
	}
	tool = normalizeActivityTool(tool)
	parts := []string{tool.Name, "failed"}
	if tool.Duration > 0 {
		parts = append(parts, "·", uitoolview.FormatParallelElapsed(tool.Duration))
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func firstLine(text string) string {
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		return text[:idx]
	}
	return text
}
