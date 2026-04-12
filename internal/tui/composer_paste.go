package tui

func (m *Model) handleComposerPaste(content string) {
	content = normalizePastedText(content)
	if content == "" {
		return
	}
	if shouldFoldPasteBlock(content) {
		m.appendPasteBlock(content)
	} else {
		m.insertTextIntoInput(content)
	}
	m.syncComposerLayout()
}

func (m *Model) removeLastPasteBlock() bool {
	if len(m.pasteBlocks) == 0 {
		return false
	}
	lastUID := m.pasteBlocks[len(m.pasteBlocks)-1].uid
	m.pasteBlocks = m.pasteBlocks[:len(m.pasteBlocks)-1]
	for i := len(m.composerParts) - 1; i >= 0; i-- {
		if m.composerParts[i].kind != composerPartPaste || m.composerParts[i].pasteUID != lastUID {
			continue
		}
		m.composerParts = append(m.composerParts[:i], m.composerParts[i+1:]...)
		break
	}
	m.promoteTrailingComposerTextToInput()
	m.syncComposerLayout()
	return true
}
