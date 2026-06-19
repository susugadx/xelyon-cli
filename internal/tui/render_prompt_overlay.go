package tui

func (m Model) renderPromptOverlay(base string) string {
	if m.prompt == nil {
		return base
	}
	return m.prompt.ViewOverlay(base, m.width, m.height)
}
