package tui

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

const (
	inputDockDraftPrefix = "  . "
	inputDockMetaPrefix  = "  + "
)

func joinChromeLines(lines []string) string {
	return strings.Join(lines, "\n")
}

func (m *Model) renderInputDockLines() []string {
	suggestionRows := m.renderSlashSuggestionRows()
	attachmentRows := m.renderAttachmentSummaryRows()
	composerRows := m.renderComposerDraftSummaryRows()
	lines := make([]string, 0, len(suggestionRows)+len(attachmentRows)+len(composerRows)+inputHeight)
	lines = append(lines, suggestionRows...)
	lines = append(lines, attachmentRows...)
	lines = append(lines, composerRows...)
	lines = append(lines, m.padLineCache, m.renderInputLine(), m.padLineCache)
	return lines
}

func (m *Model) renderInputLine() string {
	chrome := theme.Chrome
	tiView := strings.ReplaceAll(m.textInput.View(), chrome.Reset, chrome.Reset+chrome.InputBg)
	return termtext.FillANSITextWidth(chrome.InputRowMarkerFg+" "+chrome.InputPrompt+inputPrompt+chrome.InputTextFg+tiView+chrome.Reset, m.width, chrome.InputBg)
}

func (m Model) renderAttachmentSummaryRows() []string {
	summaries := m.visibleAttachmentSummaryRows()
	if len(summaries) == 0 {
		return nil
	}
	chrome := theme.Chrome
	rows := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		rows = append(rows, m.renderInputMetaStyleRow(highlightSummaryNumber(summary, chrome.InputPasteID, chrome.InputPasteFg)))
	}
	return rows
}

func (m Model) renderComposerDraftSummaryRows() []string {
	summaries := m.visibleComposerDraftSummaryRows()
	if len(summaries) == 0 {
		return nil
	}
	chrome := theme.Chrome
	rows := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		if summary.Kind == composerDraftSummaryPaste {
			rows = append(rows, m.renderInputMetaStyleRow(highlightSummaryNumber(summary.Text, chrome.InputPasteID, chrome.InputPasteFg)))
			continue
		}
		rows = append(rows, m.renderInputDraftStyleRow(summary.Text))
	}
	return rows
}

func highlightSummaryNumber(summary, numberColor, textColor string) string {
	idx := strings.LastIndex(summary, "#")
	if idx < 0 {
		return summary
	}
	return summary[:idx] + numberColor + summary[idx:] + textColor
}

func (m Model) renderInputMetaStyleRow(text string) string {
	chrome := theme.Chrome
	return termtext.FillANSITextWidth(chrome.InputMetaMarkerFg+inputDockMetaPrefix+chrome.InputPasteFg+text+chrome.Reset, m.width, chrome.InputBg)
}

func (m Model) renderInputDraftStyleRow(text string) string {
	chrome := theme.Chrome
	return termtext.FillANSITextWidth(chrome.InputRowMarkerFg+inputDockDraftPrefix+chrome.InputDraftFg+text+chrome.Reset, m.width, chrome.InputBg)
}
