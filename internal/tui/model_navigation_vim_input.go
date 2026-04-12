package tui

func (m Model) handleNavigationVimKey(s string) (Model, bool) {
	if next, handled := m.handleNavigationLineMotionKey(s); handled {
		return next, true
	}
	if next, handled := m.handleNavigationWordMotionKey(s); handled {
		return next, true
	}
	return m.handleNavigationSelectionKey(s)
}
