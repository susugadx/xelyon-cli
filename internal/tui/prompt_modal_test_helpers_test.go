package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newPromptTestModel(req ui.PromptRequest, ch chan ui.PromptResponse) Model {
	m := newSizedPromptTestModel(&stubAgent{}, 60, 16)
	updated, _ := m.Update(OpenPromptMsg{ID: 1, Request: req, Respond: ch})
	return updated.(Model)
}

func newSizedPromptTestModel(agent *stubAgent, width int, height int) Model {
	m := NewModel(agent, "")
	m.applyChatWindowSize(width, height)
	m.rebuildChrome()
	m.chromeDirty = false
	return m
}

func appendPromptTestLines(m *Model, count int) {
	for i := 0; i < count; i++ {
		m.appendContentLines(fmt.Sprintf("preview line %02d", i))
	}
	m.rebuildChrome()
	m.chromeDirty = false
}

func promptRuneKey(input string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(input)}
}
