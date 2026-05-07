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

type inputDockRowGroupKind int

const (
	inputDockRowGroupSuggestions inputDockRowGroupKind = iota
	inputDockRowGroupSelectedDetail
	inputDockRowGroupCompactChips
	inputDockRowGroupDrafts
	inputDockRowGroupTopPadding
	inputDockRowGroupInput
	inputDockRowGroupBottomPadding
)

type inputDockRowGroup struct {
	Kind  inputDockRowGroupKind
	Lines []string
}

func joinChromeLines(lines []string) string {
	return strings.Join(lines, "\n")
}

func (m *Model) renderInputDockLines() []string {
	return flattenInputDockRowGroups(m.inputDockRowGroups())
}

func (m *Model) inputDockRowGroups() []inputDockRowGroup {
	return []inputDockRowGroup{
		{Kind: inputDockRowGroupSuggestions, Lines: m.renderSlashSuggestionRows()},
		{Kind: inputDockRowGroupSelectedDetail, Lines: m.renderSelectedSlashSuggestionDetailRows()},
		{Kind: inputDockRowGroupCompactChips, Lines: m.renderCompactChipRows()},
		{Kind: inputDockRowGroupDrafts, Lines: m.renderComposerDraftSummaryRows()},
		{Kind: inputDockRowGroupTopPadding, Lines: []string{m.padLineCache}},
		{Kind: inputDockRowGroupInput, Lines: []string{m.renderInputLine()}},
		{Kind: inputDockRowGroupBottomPadding, Lines: []string{m.padLineCache}},
	}
}

func flattenInputDockRowGroups(groups []inputDockRowGroup) []string {
	lineCount := 0
	for _, group := range groups {
		lineCount += len(group.Lines)
	}
	lines := make([]string, 0, lineCount)
	for _, group := range groups {
		lines = append(lines, group.Lines...)
	}
	return lines
}

func (m *Model) renderInputLine() string {
	chrome := theme.Chrome
	tiView := strings.ReplaceAll(m.textInput.View(), chrome.Reset, chrome.Reset+chrome.InputBg)
	return termtext.FillANSITextWidth(chrome.InputRowMarkerFg+" "+chrome.InputPrompt+inputPrompt+chrome.InputTextFg+tiView+chrome.Reset, m.width, chrome.InputBg)
}

func (m Model) renderCompactChipRows() []string {
	if m.visibleCompactChipRowCount() == 0 {
		return nil
	}
	chips := m.visibleCompactChips()
	if len(chips) == 0 {
		return nil
	}
	chrome := theme.Chrome
	row := strings.Join(chips, " ")
	row = termtext.TruncateWithANSI(termtext.SanitizeSingleLineANSI(row), max(0, m.width-4))
	return []string{m.renderInputMetaStyleRow(highlightSummaryNumber(row, chrome.InputPasteID, chrome.InputPasteFg))}
}

func (m Model) renderComposerDraftSummaryRows() []string {
	summaries := m.visibleComposerDraftSummaryRows()
	if len(summaries) == 0 {
		return nil
	}
	rows := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		rows = append(rows, m.renderInputDraftStyleRow(summary.Text))
	}
	return rows
}

func (m Model) renderSelectedSlashSuggestionDetailRows() []string {
	detail := m.selectedSlashSuggestionDetailText()
	if detail == "" || m.visibleSlashSuggestionDetailRowCount() == 0 {
		return nil
	}
	chrome := theme.Chrome
	return []string{termtext.FillANSITextWidth(chrome.SuggestionBg+chrome.SuggestionSelectedDimFg+"  "+detail+chrome.Reset, m.width, chrome.SuggestionBg)}
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
