package tui

// configView は /config screen の View を構築する。
func (m Model) configView() string {
	if m.configScreen == nil {
		return "Loading..."
	}
	return m.configScreen.View(m.width, m.height)
}
