package sessionpickerscreen

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/keyinput"
)

// HandleKey は session picker のキー入力を処理する。
func (p *Screen) HandleKey(msg tea.KeyMsg) KeyResult {
	if p == nil {
		return KeyResult{}
	}

	switch {
	case msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC:
		return KeyResult{Command: CommandClose}
	case msg.Type == tea.KeyUp || (!p.filtering && msg.String() == "k"):
		p.moveSelection(-1)
	case msg.Type == tea.KeyDown || (!p.filtering && msg.String() == "j"):
		p.moveSelection(1)
	case msg.String() == "/":
		p.filtering = true
		p.filter = ""
		p.selected = 0
	case isBackspaceKey(msg):
		p.handleBackspace()
	case isEnterKey(msg):
		row, ok := p.selectedSession()
		if !ok {
			return KeyResult{}
		}
		return KeyResult{Command: CommandResume, Candidate: row}
	case p.filtering && msg.Type == tea.KeyRunes:
		p.filter += string(msg.Runes)
		p.selected = 0
	}

	p.clampSelection()
	return KeyResult{}
}

func (p *Screen) handleBackspace() {
	if !p.filtering || p.filter == "" {
		return
	}
	runes := []rune(p.filter)
	p.filter = string(runes[:len(runes)-1])
	p.selected = 0
	p.clampSelection()
}

func isEnterKey(msg tea.KeyMsg) bool {
	return keyinput.IsEnterKey(msg)
}

func isBackspaceKey(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeyBackspace || msg.Type == tea.KeyCtrlH {
		return true
	}
	s := msg.String()
	return s == "backspace" || s == "ctrl+h"
}
