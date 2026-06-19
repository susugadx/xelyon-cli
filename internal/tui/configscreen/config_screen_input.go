package configscreen

import tea "github.com/charmbracelet/bubbletea"

// Command は /config screen が root Model に要求する操作を表す。
type Command int

const (
	// CommandNone は root 側の操作が不要な入力処理を表す。
	CommandNone Command = iota
	// CommandClose は /config screen を閉じる要求を表す。
	CommandClose
	// CommandSave は設定保存を要求する。
	CommandSave
	// CommandSaveAndClose は保存成功後に /config screen を閉じる要求を表す。
	CommandSaveAndClose
	// CommandDelegateCtrlC は Ctrl+C を chat 側へ委譲する要求を表す。
	CommandDelegateCtrlC
)

type configCommand = Command

const (
	configCommandNone          = CommandNone
	configCommandClose         = CommandClose
	configCommandSave          = CommandSave
	configCommandSaveAndClose  = CommandSaveAndClose
	configCommandDelegateCtrlC = CommandDelegateCtrlC
)

// HandleKey は /config screen のキー入力を処理する。
func (cs *Screen) HandleKey(msg tea.KeyMsg, layout Layout, agentProcessing bool, providerConfigKey string) (Command, tea.Cmd) {
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

func isEnterKey(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyEnter || msg.String() == "enter"
}

func (cs *Screen) handleCtrlC(agentProcessing bool) configCommand {
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

func (cs *Screen) handleFilterKey(msg tea.KeyMsg) (configCommand, tea.Cmd) {
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

func (cs *Screen) handleConfirmKey(msg tea.KeyMsg) (configCommand, tea.Cmd) {
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

func (cs *Screen) tryClose() configCommand {
	if cs.dirty {
		cs.confirmQuit = true
		cs.confirmIdx = 0
		return configCommandNone
	}
	return configCommandClose
}
