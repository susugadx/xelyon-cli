package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/review"
)

func (rs *reviewScreen) handleKey(msg tea.KeyMsg) (reviewCommand, *review.ReviewRequest, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		return reviewCommandDelegateCtrlC, nil, nil
	}

	switch rs.mode {
	case reviewScreenPreset:
		return rs.handlePresetKey(msg)
	case reviewScreenCustom:
		return rs.handleCustomKey(msg)
	default:
		return reviewCommandNone, nil, nil
	}
}

func (rs *reviewScreen) handlePresetKey(msg tea.KeyMsg) (reviewCommand, *review.ReviewRequest, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		return reviewCommandClose, nil, nil

	case msg.Type == tea.KeyUp || msg.String() == "k":
		if rs.presetIndex > 0 {
			rs.presetIndex--
		}
		rs.clearNotice()
		return reviewCommandNone, nil, nil

	case msg.Type == tea.KeyDown || msg.String() == "j":
		if rs.presetIndex < len(reviewPresets)-1 {
			rs.presetIndex++
		}
		rs.clearNotice()
		return reviewCommandNone, nil, nil

	case isEnterKey(msg):
		preset, ok := rs.selectedPreset()
		if !ok {
			return reviewCommandNone, nil, nil
		}
		switch preset.action {
		case reviewPresetActionCurrentChanges:
			req := review.NewCurrentChangesRequest("")
			rs.customInput.Blur()
			return reviewCommandSubmit, &req, nil
		case reviewPresetActionCustomInstructions:
			rs.openCustomInput()
			return reviewCommandNone, nil, nil
		}
	}

	return reviewCommandNone, nil, nil
}

func (rs *reviewScreen) handleCustomKey(msg tea.KeyMsg) (reviewCommand, *review.ReviewRequest, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		rs.backToPreset()
		return reviewCommandNone, nil, nil

	case isEnterKey(msg):
		req := review.NewCurrentChangesRequest(rs.customInput.Value())
		rs.bodyViewport.reset()
		rs.customInput.Blur()
		return reviewCommandSubmit, &req, nil

	default:
		var cmd tea.Cmd
		rs.customInput, cmd = rs.customInput.Update(msg)
		rs.clearNotice()
		return reviewCommandNone, nil, cmd
	}
}
