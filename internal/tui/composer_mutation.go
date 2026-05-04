package tui

import (
	"unicode/utf8"

	tuicomposer "github.com/susugadx/xelyon-cli/internal/tui/composer"
)

func (m *Model) clearComposer() {
	m.composer.Clear()
	m.clearAttachments()
	m.textInput.Reset()
	m.clearSlashSuggestions()
	m.syncComposerLayout()
}

func (m *Model) clearComposerDraft() {
	m.composer.Clear()
	m.textInput.Reset()
	m.clearSlashSuggestions()
	m.syncComposerLayout()
}

func (m *Model) syncComposerLayout() {
	if m.ready {
		m.applyChatWindowSize(m.width, m.height)
		return
	}
	m.chromeDirty = true
}

func (m *Model) prependTextToInput(text string) {
	if text == "" {
		return
	}
	current := m.textInput.Value()
	cursor := m.textInput.Position()
	m.textInput.SetValue(text + current)
	m.textInput.SetCursor(utf8.RuneCountInString(text) + cursor)
}

func (m *Model) appendComposerText(text string) {
	m.composer.AppendText(text)
}

func (m *Model) insertTextIntoInput(text string) {
	if text == "" {
		return
	}
	left, right := tuicomposer.SplitRunesAt(m.textInput.Value(), m.textInput.Position())
	m.textInput.SetValue(left + text + right)
	m.textInput.SetCursor(utf8.RuneCountInString(left) + utf8.RuneCountInString(text))
}

func (m *Model) appendPasteBlock(content string) {
	if content == "" {
		return
	}
	left, right := tuicomposer.SplitRunesAt(m.textInput.Value(), m.textInput.Position())
	if left != "" {
		m.appendComposerText(left)
	}
	m.composer.AppendPasteBlock(content)
	m.textInput.SetValue(right)
	m.textInput.SetCursor(0)
}
