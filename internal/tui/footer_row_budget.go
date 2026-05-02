package tui

func (m Model) remainingFooterRowsAfterComposer() int {
	return m.maxFooterExpansionRows() - len(m.visibleComposerRows())
}

func (m Model) remainingFooterRowsAfterComposerAndAttachments() int {
	return m.remainingFooterRowsAfterComposer() - m.visibleAttachmentCount()
}
