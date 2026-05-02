package tui

import tea "github.com/charmbracelet/bubbletea"

func (m *Model) activateModalScreen(mode screenMode) {
	m.screen = mode
	m.navigationMode = false
	m.chromeDirty = true
}

func (m *Model) deactivateModalScreen(rebuildChromeNow bool) {
	m.screen = screenChat
	m.refreshStatusLine()
	m.applyChatWindowSize(m.width, m.height)
	m.textInput.Focus()
	if rebuildChromeNow {
		m.rebuildChrome()
		m.chromeDirty = false
	}
}

func (m Model) forwardMessageToChatFromModal(msg tea.Msg, modal screenMode) (tea.Model, tea.Cmd) {
	m.screen = screenChat
	updated, cmd := m.Update(msg)
	m = updated.(Model)
	m.screen = modal
	return m, cmd
}
