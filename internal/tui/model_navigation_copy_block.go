package tui

func (m *Model) copyFocusedBlock() {
	if m.focusedBlock < 0 || m.focusedBlock >= len(m.toolBlocks) {
		return
	}

	content := m.toolBlocks[m.focusedBlock].tool.Detail
	if err := m.clipboard.CopyText(content); err == nil {
		m.setCopySuccess("Copied block to clipboard")
	} else {
		m.setCopyError(err)
	}
}
