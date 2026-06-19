package tui

import (
	"fmt"
	"strings"

	tuiattachments "github.com/susugadx/xelyon-cli/internal/tui/attachments"
	tuicomposer "github.com/susugadx/xelyon-cli/internal/tui/composer"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

type composerDraftSummaryKind int

const (
	composerDraftSummaryText composerDraftSummaryKind = iota
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

func (m Model) visibleCompactChipRowCount() int {
	if len(m.visibleCompactChips()) == 0 {
		return 0
	}
	if m.remainingFooterRowsAfterComposer() <= 0 {
		return 0
	}
	return 1
}

func (m Model) visibleCompactChips() []string {
	chips := make([]string, 0, len(m.attachments)+len(m.composer.PasteBlocks))
	for i, att := range m.attachments {
		chips = append(chips, m.formatAttachmentChip(att, i+1))
	}
	for _, row := range m.composer.VisibleRows(max(1, len(m.composer.Parts))) {
		if row.Kind != tuicomposer.PartPaste {
			continue
		}
		chips = append(chips, m.formatPasteBlockChip(row.PasteBlock))
	}
	return chips
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
			text := strings.TrimSpace(m.formatComposerTextRow(row.Text))
			if text == "" {
				continue
			}
			summaries = append(summaries, composerDraftSummaryRow{
				Kind: composerDraftSummaryText,
				Text: text,
			})
		}
	}
	if len(summaries) > 1 {
		return summaries[len(summaries)-1:]
	}
	return summaries
}

func (m Model) formatAttachmentChip(att tuiattachments.Attachment, number int) string {
	return fmt.Sprintf("[%s %s #%d]", att.KindLabel(), att.Basename(), number)
}

func (m Model) formatPasteBlockChip(block tuicomposer.VisiblePasteBlock) string {
	return fmt.Sprintf("[paste %dc/%dl #%d]", block.Block.CharCount, block.Block.LineCount, block.Number)
}

func (m Model) formatComposerTextRow(text string) string {
	return termtext.SanitizeSingleLineANSI(text)
}
