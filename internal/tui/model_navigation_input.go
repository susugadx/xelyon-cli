package tui

import tea "github.com/charmbracelet/bubbletea"

// handleNavigationKey はナビゲーションモードのキー処理。
func (m Model) handleNavigationKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	var handled bool

	if next, handled := m.handleNavigationControlKey(msg); handled {
		return next, nil
	}
	m, handled = m.resolvePendingNavigationCopy(s)
	if handled {
		return m, nil
	}
	m, handled = m.resolvePendingGotoLine(s)
	if handled {
		return m, nil
	}
	if next, handled := m.handleNavigationCountPrefix(s); handled {
		return next, nil
	}
	if next, handled := m.handleNavigationModeRune(s); handled {
		return next, nil
	}

	return m.handleNavigationCommandKey(msg, s), nil
}
