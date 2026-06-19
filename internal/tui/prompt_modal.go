package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/promptmodal"
	"github.com/susugadx/xelyon-cli/internal/uiprompt"
)

func (m Model) handleOpenPromptMsg(msg OpenPromptMsg) (Model, tea.Cmd) {
	m.switchToComposerInput()
	m.prompt = promptmodal.New(msg.ID, msg.Request, msg.Respond)
	if m.screen == screenChat && m.ready {
		m.vp.gotoBottom()
		m.newOutput = false
	}
	m.chromeDirty = true
	m.rebuildChromeAfterPromptRootChange()
	return m, nil
}

func (m Model) handleCancelPromptMsg(msg CancelPromptMsg) (Model, tea.Cmd) {
	if m.prompt == nil || m.prompt.ID() != msg.ID {
		return m, nil
	}
	m.prompt = nil
	m.chromeDirty = true
	m.rebuildChromeAfterPromptRootChange()
	return m, nil
}

func (m Model) handlePromptKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.prompt == nil {
		return m, nil
	}
	result, cmd := m.prompt.HandleKey(msg)
	if result.Response != nil {
		m.finishPrompt(*result.Response)
	}
	return m, cmd
}

func (m *Model) finishPrompt(resp uiprompt.PromptResponse) {
	if m.prompt == nil {
		return
	}
	if respond := m.prompt.Respond(); respond != nil {
		select {
		case respond <- resp:
		default:
		}
	}
	m.prompt = nil
	m.chromeDirty = true
	m.rebuildChromeAfterPromptRootChange()
}

func (m *Model) rebuildChromeAfterPromptRootChange() {
	if !m.ready {
		return
	}
	m.rebuildChrome()
	m.chromeDirty = false
}
