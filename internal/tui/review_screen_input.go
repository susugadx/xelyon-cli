package tui

import tea "github.com/charmbracelet/bubbletea"

func (rs *reviewScreen) handleKey(msg tea.KeyMsg) (reviewCommand, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		return reviewCommandDelegateCtrlC, nil
	}

	switch rs.mode {
	case reviewScreenPreset:
		return rs.handlePresetKey(msg)
	case reviewScreenCustom:
		return rs.handleCustomKey(msg)
	case reviewScreenSubmitted:
		return rs.handleSubmittedKey(msg), nil
	default:
		return reviewCommandNone, nil
	}
}

func (rs *reviewScreen) handlePresetKey(msg tea.KeyMsg) (reviewCommand, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		return reviewCommandClose, nil

	case msg.Type == tea.KeyUp || msg.String() == "k":
		if rs.presetIndex > 0 {
			rs.presetIndex--
		}
		return reviewCommandNone, nil

	case msg.Type == tea.KeyDown || msg.String() == "j":
		if rs.presetIndex < len(reviewPresetLabels())-1 {
			rs.presetIndex++
		}
		return reviewCommandNone, nil

	case isEnterKey(msg):
		switch rs.presetIndex {
		case 0:
			rs.submitUncommitted("")
			return reviewCommandSubmit, nil
		case 1:
			rs.openCustomInput()
			return reviewCommandNone, nil
		}
	}

	return reviewCommandNone, nil
}

func (rs *reviewScreen) handleCustomKey(msg tea.KeyMsg) (reviewCommand, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		rs.backToPreset()
		return reviewCommandNone, nil

	case isEnterKey(msg):
		rs.submitUncommitted(rs.customInput.Value())
		return reviewCommandSubmit, nil

	default:
		var cmd tea.Cmd
		rs.customInput, cmd = rs.customInput.Update(msg)
		return reviewCommandNone, cmd
	}
}

func (rs *reviewScreen) handleSubmittedKey(msg tea.KeyMsg) reviewCommand {
	if msg.Type == tea.KeyEsc || msg.String() == "q" {
		return reviewCommandClose
	}
	return reviewCommandNone
}
