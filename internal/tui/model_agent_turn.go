package tui

const agentTurnBusyStatus = "Agent is still working"

func (m Model) agentTurnBusy() bool {
	return m.hasActiveAgentActivity() || m.conversation.IsProcessing()
}

func (m *Model) rejectAgentTurnWhileBusy() bool {
	if !m.agentTurnBusy() {
		return false
	}
	m.setTransientStatus(agentTurnBusyStatus)
	return true
}

func (m Model) appendChatAgentTurn(display string) Model {
	m.appendUserMessage(display)
	m.beginAgentActivity()
	m.chromeDirty = true
	return m
}

func (m Model) appendStartupAgentTurn(sub StartupSubmission) Model {
	if sub.UserMessage != "" {
		m.appendUserMessage(sub.UserMessage)
	}
	if sub.Cmd != nil {
		m.beginAgentActivity()
	}
	m.chromeDirty = true
	return m
}
