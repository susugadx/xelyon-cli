package tui

import tea "github.com/charmbracelet/bubbletea"

type configCommand int

const (
	configCommandNone configCommand = iota
	configCommandClose
	configCommandSave
	configCommandSaveAndClose
	configCommandDelegateCtrlC
)

func (cs *configScreen) handleKey(msg tea.KeyMsg, layout configLayout, agentProcessing bool, providerConfigKey string) (configCommand, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		return cs.handleCtrlC(agentProcessing), nil
	}
	if cs.confirmQuit {
		return cs.handleConfirmKey(msg)
	}
	if cs.filterMode {
		return cs.handleFilterKey(msg)
	}
	if cs.editMode != editNone {
		return cs.handleEditKey(msg, providerConfigKey)
	}
	return cs.handleBrowseKey(msg, layout, providerConfigKey)
}

func (cs *configScreen) handleCtrlC(agentProcessing bool) configCommand {
	if agentProcessing {
		return configCommandDelegateCtrlC
	}
	if cs.confirmQuit {
		return configCommandNone
	}
	if cs.dirty {
		return cs.tryClose()
	}
	return configCommandDelegateCtrlC
}

func (cs *configScreen) handleFilterKey(msg tea.KeyMsg) (configCommand, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		cs.filterMode = false
		cs.filterText = ""
		cs.filterInput.Blur()
		cs.fieldIndex = 0
		cs.fieldScroll = 0
		return configCommandNone, nil

	case isEnterKey(msg):
		cs.filterText = cs.filterInput.Value()
		cs.filterMode = false
		cs.filterInput.Blur()
		cs.fieldIndex = 0
		cs.fieldScroll = 0
		return configCommandNone, nil

	default:
		var cmd tea.Cmd
		cs.filterInput, cmd = cs.filterInput.Update(msg)
		cs.filterText = cs.filterInput.Value()
		cs.fieldIndex = 0
		cs.fieldScroll = 0
		return configCommandNone, cmd
	}
}

func (cs *configScreen) handleConfirmKey(msg tea.KeyMsg) (configCommand, tea.Cmd) {
	s := msg.String()

	switch {
	case msg.Type == tea.KeyEsc:
		cs.confirmQuit = false
		return configCommandNone, nil

	case msg.Type == tea.KeyUp || s == "k":
		if cs.confirmIdx > 0 {
			cs.confirmIdx--
		}
		return configCommandNone, nil

	case msg.Type == tea.KeyDown || s == "j":
		if cs.confirmIdx < 2 {
			cs.confirmIdx++
		}
		return configCommandNone, nil

	case isEnterKey(msg):
		switch cs.confirmIdx {
		case 0:
			return configCommandSaveAndClose, nil
		case 1:
			if cs.saveStatus == statusSaving {
				return configCommandNone, nil
			}
			cs.confirmQuit = false
			return configCommandClose, nil
		case 2:
			cs.confirmQuit = false
			return configCommandNone, nil
		}
	}
	return configCommandNone, nil
}

func (cs *configScreen) tryClose() configCommand {
	if cs.dirty {
		cs.confirmQuit = true
		cs.confirmIdx = 0
		return configCommandNone
	}
	return configCommandClose
}
