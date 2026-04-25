package tui

import (
	"fmt"

	tuicomposer "github.com/susugadx/xelyon-cli/internal/tui/composer"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

func (m Model) maxVisibleComposerRows() int {
	if len(m.composer.Parts) == 0 {
		return 0
	}
	if m.height <= 0 {
		return len(m.composer.Parts)
	}
	return max(0, m.height-statusBarHeight-inputHeight-1)
}

func (m Model) visibleComposerRows() []tuicomposer.VisibleRow {
	return m.composer.VisibleRows(m.maxVisibleComposerRows())
}

func (m Model) formatPasteBlockSummary(block tuicomposer.VisiblePasteBlock) string {
	return fmt.Sprintf("[Pasted Content %d chars, %d lines] #%d", block.Block.CharCount, block.Block.LineCount, block.Number)
}

func (m Model) formatComposerTextRow(text string) string {
	return termtext.SanitizeSingleLineANSI(text)
}
