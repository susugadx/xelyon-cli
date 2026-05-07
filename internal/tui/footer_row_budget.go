package tui

func (m Model) remainingFooterRowsAfterComposer() int {
	return m.maxFooterExpansionRows() - len(m.visibleComposerDraftSummaryRows())
}

func (m Model) remainingFooterRowsAfterComposerAndAttachments() int {
	return m.remainingFooterRowsAfterComposer() - m.visibleCompactChipRowCount()
}
