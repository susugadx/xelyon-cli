package tui

import tuicomposer "github.com/susugadx/xelyon-cli/internal/tui/composer"

func (m *Model) handleComposerPaste(content string) {
	content = tuicomposer.NormalizePastedText(content)
	if content == "" {
		return
	}
	if tuicomposer.ShouldFoldPasteBlock(content) {
		m.appendPasteBlock(content)
	} else {
		m.insertTextIntoInput(content)
	}
	m.syncComposerLayout()
	m.refreshSlashSuggestions()
}

func (m *Model) removeLastPasteBlock() bool {
	text, ok := m.composer.RemoveLastPasteBlock()
	if !ok {
		return false
	}
	m.prependTextToInput(text)
	m.syncComposerLayout()
	m.refreshSlashSuggestions()
	return true
}
