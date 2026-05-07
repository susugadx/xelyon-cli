package tui

import (
	"fmt"

	tuicomposer "github.com/susugadx/xelyon-cli/internal/tui/composer"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

type composerDraftSummaryKind int

const (
	composerDraftSummaryText composerDraftSummaryKind = iota
	composerDraftSummaryPaste
)

type composerDraftSummaryRow struct {
	Kind composerDraftSummaryKind
	Text string
}

func (m Model) maxVisibleComposerRows() int {
	if len(m.composer.Parts) == 0 {
		return 0
	}
	if m.height <= 0 {
		return len(m.composer.Parts)
	}
	return m.maxFooterExpansionRows()
}

func (m Model) visibleComposerRows() []tuicomposer.VisibleRow {
	return m.composer.VisibleRows(m.maxVisibleComposerRows())
}

func (m Model) visibleAttachmentSummaryRows() []string {
	start, end := m.visibleAttachmentRange()
	if start >= end {
		return nil
	}
	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		rows = append(rows, m.formatAttachmentSummary(m.attachments[i], i+1))
	}
	return rows
}

func (m Model) visibleComposerDraftSummaryRows() []composerDraftSummaryRow {
	rows := m.visibleComposerRows()
	if len(rows) == 0 {
		return nil
	}
	summaries := make([]composerDraftSummaryRow, 0, len(rows))
	for _, row := range rows {
		switch row.Kind {
		case tuicomposer.PartText:
			summaries = append(summaries, composerDraftSummaryRow{
				Kind: composerDraftSummaryText,
				Text: m.formatComposerTextRow(row.Text),
			})
		case tuicomposer.PartPaste:
			summaries = append(summaries, composerDraftSummaryRow{
				Kind: composerDraftSummaryPaste,
				Text: m.formatPasteBlockSummary(row.PasteBlock),
			})
		}
	}
	return summaries
}

func (m Model) formatPasteBlockSummary(block tuicomposer.VisiblePasteBlock) string {
	return fmt.Sprintf("[Pasted Content %d chars, %d lines] #%d", block.Block.CharCount, block.Block.LineCount, block.Number)
}

func (m Model) formatAttachmentSummary(att composerAttachment, number int) string {
	return fmt.Sprintf("[Attached %s %s] #%d", att.kindLabel(), att.basename(), number)
}

func (m Model) formatComposerTextRow(text string) string {
	return termtext.SanitizeSingleLineANSI(text)
}
