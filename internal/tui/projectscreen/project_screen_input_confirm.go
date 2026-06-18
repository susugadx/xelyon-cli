package projectscreen

import tea "github.com/charmbracelet/bubbletea"

func (ps *Screen) handleConfirmKey(msg tea.KeyMsg) Command {
	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc:
		ps.confirmQuit = false
		ps.pendingClose = false
		return CommandNone
	case msg.Type == tea.KeyUp || s == "k":
		if ps.confirmIdx > 0 {
			ps.confirmIdx--
		}
	case msg.Type == tea.KeyDown || s == "j":
		if ps.confirmIdx < 2 {
			ps.confirmIdx++
		}
	case isEnterKey(msg):
		switch ps.confirmIdx {
		case 0:
			return CommandSaveAndClose
		case 1:
			if ps.saveInFlight {
				return CommandNone
			}
			ps.confirmQuit = false
			return CommandClose
		case 2:
			ps.confirmQuit = false
			ps.pendingClose = false
		}
	}
	return CommandNone
}
