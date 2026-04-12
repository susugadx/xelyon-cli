package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleMouseSelection はマウスイベントの選択処理を行う。
// transcript viewport 領域でのみ選択を扱い、input dock 上では無視する。
func (m *Model) handleMouseSelection(msg tea.MouseMsg) tea.Cmd {
	switch msg.Action {
	case tea.MouseActionPress:
		return m.handleMouseSelectionPress(msg)
	case tea.MouseActionMotion:
		return m.handleMouseSelectionMotion(msg)
	case tea.MouseActionRelease:
		return m.handleMouseSelectionRelease(msg)
	}
	return nil
}
