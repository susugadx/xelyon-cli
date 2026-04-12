package tui

import tea "github.com/charmbracelet/bubbletea"

func newStreamTestModel(width, height int) (Model, *stubAgent) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return updated.(Model), agent
}
