package tui

import tea "github.com/charmbracelet/bubbletea"

func (m *Model) resetStreamingState() {
	m.streamingActive = false
	m.streamCursorCol = 0
	m.streamActiveANSI = ""
	m.streamPendingANSI = ""
}

func (m *Model) handleAppendMessageMsg(msg AppendMessageMsg) tea.Cmd {
	m.resetStreamingState()
	return m.appendMessage(msg.Message)
}

func (m *Model) handleAppendToolResultMsg(msg AppendToolResultMsg) tea.Cmd {
	m.resetStreamingState()
	if m.hasActiveAgentActivity() {
		return m.upsertAgentActivityTool(msg.Tool)
	}
	return m.appendToolResult(msg.Tool)
}

func (m *Model) handleStreamTextMsg(msg StreamTextMsg) tea.Cmd {
	cmd := m.appendStreamText(msg.Text)
	if msg.Done {
		m.resetStreamingState()
		m.refreshStatusLine()
	}
	return cmd
}

func (m *Model) handleUpdateStatusMsg(msg UpdateStatusMsg) {
	m.resetStreamingState()
	m.statusLine = msg.Line
	m.statusSnapshot.LegacyLine = msg.Line
	m.statusSnapshot.Mode = msg.Line
	m.chromeDirty = true
}

func (m *Model) handleAgentDoneMsg(msg AgentDoneMsg) {
	m.resetStreamingState()
	m.refreshStatusLine()
	m.finishAgentActivity(msg.Error, msg.ErrorKind)
}

func (m Model) handleStreamMessage(msg tea.Msg) (Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case AppendMessageMsg:
		return m, m.handleAppendMessageMsg(msg), true
	case AppendToolResultMsg:
		return m, m.handleAppendToolResultMsg(msg), true
	case StreamTextMsg:
		return m, m.handleStreamTextMsg(msg), true
	case UpdateStatusMsg:
		m.handleUpdateStatusMsg(msg)
		return m, nil, true
	case AgentDoneMsg:
		m.handleAgentDoneMsg(msg)
		return m, nil, true
	default:
		return m, nil, false
	}
}
