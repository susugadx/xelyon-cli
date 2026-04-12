package tui

import (
	"strings"
	"unicode/utf8"
)

func (m *Model) clearComposer() {
	m.composerParts = nil
	m.pasteBlocks = nil
	m.nextPasteUID = 0
	m.textInput.Reset()
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
	if text == "" {
		return
	}
	if n := len(m.composerParts); n > 0 && m.composerParts[n-1].kind == composerPartText {
		m.composerParts[n-1].text += text
		return
	}
	m.composerParts = append(m.composerParts, composerPart{
		kind: composerPartText,
		text: text,
	})
}

func (m *Model) promoteTrailingComposerTextToInput() {
	if len(m.composerParts) == 0 {
		return
	}
	tail := make([]string, 0, len(m.composerParts))
	for len(m.composerParts) > 0 {
		last := m.composerParts[len(m.composerParts)-1]
		if last.kind != composerPartText {
			break
		}
		tail = append(tail, last.text)
		m.composerParts = m.composerParts[:len(m.composerParts)-1]
	}
	if len(tail) == 0 {
		return
	}
	var builder strings.Builder
	for i := len(tail) - 1; i >= 0; i-- {
		builder.WriteString(tail[i])
	}
	m.prependTextToInput(builder.String())
}

func (m *Model) insertTextIntoInput(text string) {
	if text == "" {
		return
	}
	left, right := splitRunesAt(m.textInput.Value(), m.textInput.Position())
	m.textInput.SetValue(left + text + right)
	m.textInput.SetCursor(utf8.RuneCountInString(left) + utf8.RuneCountInString(text))
}

func (m *Model) appendPasteBlock(content string) {
	if content == "" {
		return
	}
	left, right := splitRunesAt(m.textInput.Value(), m.textInput.Position())
	if left != "" {
		m.appendComposerText(left)
	}
	m.nextPasteUID++
	block := pasteBlock{
		uid:       m.nextPasteUID,
		content:   content,
		charCount: utf8.RuneCountInString(content),
		lineCount: strings.Count(content, "\n") + 1,
	}
	m.pasteBlocks = append(m.pasteBlocks, block)
	m.composerParts = append(m.composerParts, composerPart{
		kind:     composerPartPaste,
		pasteUID: block.uid,
	})
	m.textInput.SetValue(right)
	m.textInput.SetCursor(0)
}
