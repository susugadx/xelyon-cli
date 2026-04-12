package tui

import "fmt"

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

	maxVisible := m.maxVisibleComposerRows()
	if maxVisible <= 0 {
		return nil
	}
	if len(rows) <= maxVisible {
		return rows
	}
	return rows[len(rows)-maxVisible:]
}

func (m Model) formatPasteBlockSummary(block visiblePasteBlock) string {
	return fmt.Sprintf("[Pasted Content %d chars, %d lines] #%d", block.block.charCount, block.block.lineCount, block.number)
}

func (m Model) formatComposerTextRow(text string) string {
	return sanitizeSingleLineANSI(text)
}
