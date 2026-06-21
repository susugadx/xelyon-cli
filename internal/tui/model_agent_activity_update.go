package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

func (m *Model) beginAgentActivity() tea.Cmd {
	return m.beginAgentActivityWithOptions(agentActivityOptions{})
}

func (m *Model) beginAgentActivityWithOptions(opts agentActivityOptions) tea.Cmd {
	if m.hasActiveAgentActivity() {
		m.setTransientStatus(agentTurnBusyStatus)
		return nil
	}

	now := time.Now()
	m.statusSnapshot = m.conversation.StatusSnapshot()
	m.statusLine = m.statusSnapshot.LegacyLine
	if m.statusLine == "" {
		m.statusLine = m.conversation.GetStatusLine()
		m.statusSnapshot.LegacyLine = m.statusLine
	}
	m.agentActivity = newAgentActivityState(now, opts)
	m.chromeDirty = true
	return m.appendTrackedBlockLines(&m.agentActivity.block, m.buildAgentActivityLines(now))
}

func (m Model) hasActiveAgentActivity() bool {
	return m.agentActivity.active && m.agentActivity.block.lineCount > 0
}

func (m *Model) updateAgentActivitySnapshot(now time.Time) {
	if !m.hasActiveAgentActivity() {
		return
	}
	m.updateTrackedBlockLinesFollowing(&m.agentActivity.block, m.buildAgentActivityLines(now))
	m.chromeDirty = true
}

func (m *Model) finishAgentActivity(err error, kind AgentErrorKind) {
	if !m.hasActiveAgentActivity() {
		return
	}

	now := time.Now()
	m.agentActivity.active = false
	m.agentActivity.finishedAt = now
	switch {
	case err != nil:
		m.agentActivity.status = agentActivityStatusBlocked
		m.agentActivity.errorText = termtext.SanitizeSingleLineANSI(err.Error())
		m.agentActivity.errorKind = AgentErrorKindFromError(err, kind)
	default:
		m.agentActivity.status = agentActivityStatusDone
		m.agentActivity.errorText = ""
		m.agentActivity.errorKind = AgentErrorUnknown
	}

	m.updateTrackedBlockLinesFollowing(&m.agentActivity.block, m.buildAgentActivityLines(now))
	m.chromeDirty = true
}
