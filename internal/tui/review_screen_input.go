package tui

import tea "github.com/charmbracelet/bubbletea"

func (rs *reviewScreen) handleKey(msg tea.KeyMsg, bodyBounds reviewBodyScrollBounds) (reviewCommand, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		return reviewCommandDelegateCtrlC, nil
	}

	switch rs.mode {
	case reviewScreenPreset:
		return rs.handlePresetKey(msg)
	case reviewScreenCustom:
		return rs.handleCustomKey(msg)
	case reviewScreenSubmitted:
		return rs.handleSubmittedKey(msg, bodyBounds), nil
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
			rs.submitCurrentChanges("")
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
		rs.submitCurrentChanges(rs.customInput.Value())
		return reviewCommandSubmit, nil

	default:
		var cmd tea.Cmd
		rs.customInput, cmd = rs.customInput.Update(msg)
		return reviewCommandNone, cmd
	}
}

func (rs *reviewScreen) handleMouse(msg tea.MouseMsg, bodyBounds reviewBodyScrollBounds) bool {
	if rs.mode != reviewScreenSubmitted {
		return false
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		rs.bodyViewport.scrollUp(3, bodyBounds)
		return true
	case tea.MouseButtonWheelDown:
		rs.bodyViewport.scrollDown(3, bodyBounds)
		return true
	default:
		return false
	}
}

func (rs *reviewScreen) handleSubmittedKey(msg tea.KeyMsg, bodyBounds reviewBodyScrollBounds) reviewCommand {
	switch {
	case msg.Type == tea.KeyEsc || msg.String() == "q":
		if rs.runState == reviewRunRunning {
			rs.cancelRunningReview()
			return reviewCommandNone
		}
		return reviewCommandClose
	case msg.Type == tea.KeyUp || msg.String() == "k":
		rs.bodyViewport.scrollUp(1, bodyBounds)
	case msg.Type == tea.KeyDown || msg.String() == "j":
		rs.bodyViewport.scrollDown(1, bodyBounds)
	case msg.Type == tea.KeyPgUp:
		rs.bodyViewport.scrollUp(bodyBounds.pageSize(), bodyBounds)
	case msg.Type == tea.KeyPgDown:
		rs.bodyViewport.scrollDown(bodyBounds.pageSize(), bodyBounds)
	case msg.Type == tea.KeyHome || msg.String() == "g":
		rs.bodyViewport.gotoTop()
	case msg.Type == tea.KeyEnd || msg.String() == "G":
		rs.bodyViewport.gotoBottom(bodyBounds)
	}
	return reviewCommandNone
}
