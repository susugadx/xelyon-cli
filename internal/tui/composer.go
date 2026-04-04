package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const pasteBlockFoldThreshold = 160

type composerPartKind int

const (
	composerPartText composerPartKind = iota
	composerPartPaste
)

type composerPart struct {
	kind     composerPartKind
	text     string
	pasteUID int
}

type pasteBlock struct {
	uid       int
	content   string
	charCount int
	lineCount int
}

type visiblePasteBlock struct {
	block  pasteBlock
	number int
}

type visibleComposerRow struct {
	kind       composerPartKind
	text       string
	pasteBlock visiblePasteBlock
}

func normalizePastedText(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}

func shouldFoldPasteBlock(content string) bool {
	content = normalizePastedText(content)
	return strings.Contains(content, "\n") || utf8.RuneCountInString(content) >= pasteBlockFoldThreshold
}

func splitRunesAt(s string, pos int) (string, string) {
	runes := []rune(s)
	if pos < 0 {
		pos = 0
	}
	if pos > len(runes) {
		pos = len(runes)
	}
	return string(runes[:pos]), string(runes[pos:])
}

func (m Model) hasComposerDraft() bool {
	return strings.TrimSpace(m.textInput.Value()) != "" || len(m.composerParts) > 0 || len(m.pasteBlocks) > 0
}

func (m Model) hasSubmittableComposerContent() bool {
	if strings.TrimSpace(m.textInput.Value()) != "" {
		return true
	}
	for _, part := range m.composerParts {
		switch part.kind {
		case composerPartText:
			if strings.TrimSpace(part.text) != "" {
				return true
			}
		case composerPartPaste:
			block, ok := m.findPasteBlock(part.pasteUID)
			if ok && block.content != "" {
				return true
			}
		}
	}
	return false
}

func (m Model) hasFoldedPasteBlocks() bool {
	return len(m.pasteBlocks) > 0
}

func (m Model) isPlainComposerInput() bool {
	return len(m.composerParts) == 0 && len(m.pasteBlocks) == 0
}

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

func (m Model) findPasteBlock(uid int) (pasteBlock, bool) {
	for _, block := range m.pasteBlocks {
		if block.uid == uid {
			return block, true
		}
	}
	return pasteBlock{}, false
}

func (m Model) buildComposerPayload() string {
	var builder strings.Builder
	for _, part := range m.composerParts {
		switch part.kind {
		case composerPartText:
			builder.WriteString(part.text)
		case composerPartPaste:
			block, ok := m.findPasteBlock(part.pasteUID)
			if ok {
				builder.WriteString(block.content)
			}
		}
	}
	builder.WriteString(m.textInput.Value())
	return builder.String()
}

func (m Model) maxVisibleComposerRows() int {
	if len(m.composerParts) == 0 {
		return 0
	}
	if m.height <= 0 {
		return len(m.composerParts)
	}
	return max(0, m.height-statusBarHeight-inputHeight-1)
}

func (m Model) visibleComposerRows() []visibleComposerRow {
	if len(m.composerParts) == 0 {
		return nil
	}
	rows := make([]visibleComposerRow, 0, len(m.composerParts))
	pasteNumber := 0
	for _, part := range m.composerParts {
		switch part.kind {
		case composerPartText:
			if part.text == "" {
				continue
			}
			rows = append(rows, visibleComposerRow{
				kind: composerPartText,
				text: part.text,
			})
		case composerPartPaste:
			block, ok := m.findPasteBlock(part.pasteUID)
			if !ok {
				continue
			}
			pasteNumber++
			rows = append(rows, visibleComposerRow{
				kind: composerPartPaste,
				pasteBlock: visiblePasteBlock{
					block:  block,
					number: pasteNumber,
				},
			})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	start := 0
	maxVisible := m.maxVisibleComposerRows()
	if maxVisible <= 0 {
		return nil
	}
	if len(rows) > maxVisible {
		start = len(rows) - maxVisible
	}
	return rows[start:]
}

func (m Model) formatPasteBlockSummary(block visiblePasteBlock) string {
	return fmt.Sprintf("[Pasted Content %d chars, %d lines] #%d", block.block.charCount, block.block.lineCount, block.number)
}

func (m Model) formatComposerTextRow(text string) string {
	return sanitizeSingleLineANSI(text)
}
