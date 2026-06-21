// Package keyinput は TUI 全体で共有するキー判定を提供する。
package keyinput

import tea "github.com/charmbracelet/bubbletea"

// IsEnterKey は Enter キーかどうかを判定する。
//
// WSL2/Windows Terminal 環境で tea.KeyEnter が正しく認識されず、
// raw CR/LF として届く場合も Enter として扱う。
func IsEnterKey(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeyEnter || msg.Type == tea.KeyCtrlJ {
		return true
	}
	s := msg.String()
	return s == "enter" || s == "\r" || s == "\n"
}
