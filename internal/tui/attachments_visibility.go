package tui

func (m Model) visibleAttachmentStartIndex() int {
	start, _ := m.visibleAttachmentRange()
	return start
}

func (m Model) visibleAttachmentRange() (start, end int) {
	if len(m.attachments) == 0 {
		return 0, 0
	}
	maxVisible := m.maxVisibleAttachmentRows()
	if maxVisible <= 0 {
		return len(m.attachments), len(m.attachments)
	}
	if len(m.attachments) <= maxVisible {
		return 0, len(m.attachments)
	}
	return len(m.attachments) - maxVisible, len(m.attachments)
}

func (m Model) visibleAttachmentCount() int {
	start, end := m.visibleAttachmentRange()
	return end - start
}

func (m Model) visibleAttachmentNumber(visibleIndex int) int {
	start, _ := m.visibleAttachmentRange()
	return start + visibleIndex + 1
}

func (m Model) maxVisibleAttachmentRows() int {
	if len(m.attachments) == 0 {
		return 0
	}
	if m.height <= 0 {
		return len(m.attachments)
	}
	remaining := m.remainingFooterRowsAfterComposer()
	if remaining <= 0 {
		return 0
	}
	return remaining
}

func (m Model) visibleAttachments() []composerAttachment {
	start, end := m.visibleAttachmentRange()
	if start >= end {
		return nil
	}
	out := make([]composerAttachment, end-start)
	copy(out, m.attachments[start:end])
	return out
}
