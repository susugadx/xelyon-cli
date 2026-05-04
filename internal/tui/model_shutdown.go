package tui

import tea "github.com/charmbracelet/bubbletea"

// beginQuit は TUI 終了時に必要な state/resource cleanup を一元化する。
// 現在は composer draft（特に temporary attachment）解放と agent cleanup を担当する。
func (m *Model) beginQuit() (tea.Model, tea.Cmd) {
	m.clearComposer()
	m.quitting = true
	m.conversation.Cleanup()
	return *m, tea.Quit
}

func (m *Model) recordHandledCommand(input string) {
	m.resetComposerInput()
	m.appendUserMessage(input)
}
