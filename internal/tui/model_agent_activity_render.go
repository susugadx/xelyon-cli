package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
	"github.com/susugadx/xelyon-cli/internal/uitoolview"
)

func (m Model) buildAgentActivityLines(now time.Time) []string {
	activity := m.agentActivity
	if activity.startedAt.IsZero() {
		activity.startedAt = now
	}

	lines := []string{m.renderAgentActivityHeader(activity, now)}
	if !activity.active {
		switch activity.status {
		case agentActivityStatusBlocked:
			return append(lines, m.renderAgentActivityBlockedSummary(activity)...)
		default:
			return append(lines, m.renderAgentActivityDoneSummary(activity))
		}
	}

	if activity.status == agentActivityStatusBlocked {
		for _, entry := range activity.tools {
			lines = append(lines, renderAgentActivityToolLine(entry.tool))
		}
		lines = append(lines, renderAgentActivityActionNeededLine(activity.errorKind))
		return lines
	}

	lines = append(lines, m.renderAgentActivityWorkingLine())
	for _, entry := range activity.tools {
		lines = append(lines, renderAgentActivityToolLine(entry.tool))
	}
	return lines
}

func (m Model) renderAgentActivityHeader(activity agentActivityState, now time.Time) string {
	palette := theme.Activity
	status := activity.status
	if status == "" {
		status = agentActivityStatusWorking
	}
	title := activity.title
	if title == "" {
		title = "agent"
	}

	elapsed := now.Sub(activity.startedAt)
	if !activity.finishedAt.IsZero() {
		elapsed = activity.finishedAt.Sub(activity.startedAt)
	}
	if elapsed < 0 {
		elapsed = 0
	}

	elapsedText := formatAgentFinalElapsed(elapsed)
	if activity.active {
		elapsedText = formatAgentClockElapsed(elapsed)
	}
	return palette.Header + fmt.Sprintf("── %s · %s · %s ──", title, status, elapsedText) + palette.Reset
}

func (m Model) renderAgentActivityWorkingLine() string {
	palette := theme.Activity
	snapshot := sanitizeStatusSnapshot(m.statusSnapshot)
	workingText := m.agentActivity.workingText
	if workingText == "" {
		workingText = "working"
	}
	parts := []string{workingText}
	if !m.agentActivity.hideStatus {
		if providerModel := providerModelStatusText(snapshot, ""); providerModel != "" {
			parts = append(parts, providerModel)
		}
		if snapshot.Tokens != "" {
			parts = append(parts, snapshot.Tokens+" tok")
		}
		if snapshot.Cost != "" {
			parts = append(parts, snapshot.Cost)
		}
	}

	scanner := m.spinner.View()
	if strings.TrimSpace(termtext.StripANSI(scanner)) == "" {
		scanner = "•"
	}
	return "│ " + palette.Scanner + scanner + palette.Reset + " " + palette.Dim + strings.Join(parts, " · ") + palette.Reset
}

func (m Model) renderAgentActivityDoneSummary(activity agentActivityState) string {
	palette := theme.Activity
	var parts []string
	if activity.doneText != "" {
		parts = append(parts, activity.doneText)
		if len(activity.tools) > 0 {
			parts = append(parts, fmt.Sprintf("%d tools", len(activity.tools)))
		}
	} else {
		parts = append(parts, fmt.Sprintf("%d tools", len(activity.tools)))
	}
	if !activity.hideStatus {
		snapshot := sanitizeStatusSnapshot(m.statusSnapshot)
		if snapshot.Tokens != "" {
			parts = append(parts, snapshot.Tokens+" tok")
		}
		if snapshot.Cost != "" {
			parts = append(parts, snapshot.Cost)
		}
	}
	return "│ " + palette.Success + "✓ " + palette.Reset + palette.Dim + strings.Join(parts, " · ") + palette.Reset
}

func (m Model) renderAgentActivityBlockedSummary(activity agentActivityState) []string {
	errorText := termtext.SanitizeSingleLineANSI(strings.TrimSpace(activity.errorText))
	if errorText == "" {
		errorText = "agent blocked"
	}
	kind := normalizeAgentErrorKind(activity.errorKind)
	color := agentActivityErrorColor(kind)
	return []string{
		"│ " + color + "✕ " + agentActivityErrorLabel(kind) + " " + errorText + theme.Activity.Reset,
		renderAgentActivityActionNeededLine(kind),
	}
}

func renderAgentActivityActionNeededLine(kind AgentErrorKind) string {
	palette := theme.Activity
	kind = normalizeAgentErrorKind(kind)
	return "│ " + agentActivityErrorColor(kind) + "! " + palette.Reset + palette.Dim + agentActivityErrorHint(kind) + palette.Reset
}

func renderAgentActivityToolLine(tool ToolResult) string {
	text := renderAgentActivityToolText(tool)
	palette := theme.Activity
	switch tool.Status {
	case ToolStatusRunning:
		return "│ " + palette.Running + text + palette.Reset
	case ToolStatusError:
		return "│ " + palette.ErrorTool + text + palette.Reset
	default:
		return "│ " + palette.Success + text + palette.Reset
	}
}

func renderAgentActivityToolText(tool ToolResult) string {
	tool = normalizeActivityTool(tool)
	switch tool.Status {
	case ToolStatusRunning:
		return strings.Join(nonEmptyStrings("● running", tool.Name, tool.Target), " ")
	case ToolStatusError:
		parts := []string{"✕", agentActivityErrorLabel(AgentErrorTool), tool.Name, "failed"}
		if tool.Duration > 0 {
			parts = append(parts, "·", uitoolview.FormatParallelElapsed(tool.Duration))
		}
		return strings.Join(parts, " ")
	default:
		parts := nonEmptyStrings("✓", tool.Name, tool.Target)
		if tool.Duration > 0 {
			parts = append(parts, "·", uitoolview.FormatParallelElapsed(tool.Duration))
		}
		return strings.Join(parts, " ")
	}
}

func agentActivityErrorLabel(kind AgentErrorKind) string {
	switch normalizeAgentErrorKind(kind) {
	case AgentErrorProvider:
		return "[provider error]"
	case AgentErrorTool:
		return "[tool error]"
	case AgentErrorValidation:
		return "[validation error]"
	case AgentErrorStartup:
		return "[startup error]"
	default:
		return "[error]"
	}
}

func agentActivityErrorHint(kind AgentErrorKind) string {
	switch normalizeAgentErrorKind(kind) {
	case AgentErrorProvider:
		return "check provider/network and retry"
	case AgentErrorTool:
		return "inspect tool output or adjust the request"
	case AgentErrorValidation:
		return "fix the input and retry"
	case AgentErrorStartup:
		return "check startup command output and retry"
	default:
		return "user action may be needed"
	}
}

func agentActivityErrorColor(kind AgentErrorKind) string {
	switch normalizeAgentErrorKind(kind) {
	case AgentErrorTool:
		return theme.Activity.ErrorTool
	case AgentErrorValidation:
		return theme.Activity.ErrorValidation
	case AgentErrorStartup:
		return theme.Activity.ErrorStartup
	case AgentErrorProvider:
		return theme.Activity.ErrorProvider
	default:
		return theme.Activity.Error
	}
}

func nonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func formatAgentClockElapsed(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSeconds := int(d.Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func formatAgentFinalElapsed(d time.Duration) string {
	switch {
	case d < 0:
		return "0ms"
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	default:
		totalSeconds := int(d.Seconds())
		return fmt.Sprintf("%dm%02ds", totalSeconds/60, totalSeconds%60)
	}
}
