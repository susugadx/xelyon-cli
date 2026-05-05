package tui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) updateWithPromptOpen(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.routePromptOpenKeyMsg(msg)
	case tea.MouseMsg:
		if isPromptBackgroundWheelMsg(msg) {
			return m.forwardMessageBehindPrompt(msg)
		}
		return m, nil
	default:
		return m.forwardMessageBehindPrompt(msg)
	}
}

func (m Model) routePromptOpenKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	if shouldHandlePromptKeyAsGlobalInterrupt(m, msg) {
		updated, cmd := m.handleCtrlC()
		return updated.(Model), cmd
	}
	if shouldForwardPromptKeyBehindPrompt(msg) {
		return m.forwardMessageBehindPrompt(msg)
	}
	return m.handlePromptKeyMsg(msg)
}

func (m Model) forwardMessageBehindPrompt(msg tea.Msg) (Model, tea.Cmd) {
	activePrompt := m.prompt
	m.prompt = nil

	updated, cmd := m.Update(msg)
	next := updated.(Model)
	next.prompt = activePrompt
	return next, cmd
}

func shouldForwardPromptKeyBehindPrompt(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyPgUp || msg.Type == tea.KeyPgDown
}

func shouldHandlePromptKeyAsGlobalInterrupt(m Model, msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyCtrlC && m.conversation.IsProcessing()
}

func isPromptBackgroundWheelMsg(msg tea.MouseMsg) bool {
	return msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown
}
