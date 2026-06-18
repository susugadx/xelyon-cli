package reviewscreen

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/review"
)

// HandleKey は review screen のキー入力を処理し、root Model 側の操作要求を返す。
func (rs *Screen) HandleKey(msg tea.KeyMsg) (Command, *review.ReviewRequest, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		return CommandDelegateCtrlC, nil, nil
	}

	switch rs.mode {
	case ModePreset:
		return rs.handlePresetKey(msg)
	case ModeCustom:
		return rs.handleCustomKey(msg)
	default:
		return CommandNone, nil, nil
	}
}

func (rs *Screen) handlePresetKey(msg tea.KeyMsg) (Command, *review.ReviewRequest, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		return CommandClose, nil, nil

	case msg.Type == tea.KeyUp || msg.String() == "k":
		if rs.presetIndex > 0 {
			rs.presetIndex--
		}
		rs.clearNotice()
		return CommandNone, nil, nil

	case msg.Type == tea.KeyDown || msg.String() == "j":
		if rs.presetIndex < len(reviewPresets)-1 {
			rs.presetIndex++
		}
		rs.clearNotice()
		return CommandNone, nil, nil

	case isEnterKey(msg):
		preset, ok := rs.selectedPreset()
		if !ok {
			return CommandNone, nil, nil
		}
		switch preset.action {
		case reviewPresetActionCurrentChanges:
			req := review.NewCurrentChangesRequest("")
			rs.customInput.Blur()
			return CommandSubmit, &req, nil
		case reviewPresetActionCustomInstructions:
			rs.openCustomInput()
			return CommandNone, nil, nil
		}
	}

	return CommandNone, nil, nil
}

func (rs *Screen) handleCustomKey(msg tea.KeyMsg) (Command, *review.ReviewRequest, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		rs.backToPreset()
		return CommandNone, nil, nil

	case isEnterKey(msg):
		req := review.NewCurrentChangesRequest(rs.customInput.Value())
		rs.bodyViewport.reset()
		rs.customInput.Blur()
		return CommandSubmit, &req, nil

	default:
		var cmd tea.Cmd
		rs.customInput, cmd = rs.customInput.Update(msg)
		rs.clearNotice()
		return CommandNone, nil, cmd
	}
}

func isEnterKey(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeyEnter {
		return true
	}
	s := msg.String()
	return s == "enter" || s == "\r" || s == "\n"
}
