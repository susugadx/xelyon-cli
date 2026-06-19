package tui

import (
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/slash"
	"github.com/susugadx/xelyon-cli/internal/tui/slashsuggestions"
)

func (m Model) currentSlashSuggestionPrefix() (string, bool) {
	if !m.isPlainComposerInput() {
		return "", false
	}
	input := m.textInput.Value()
	if input == "" || !strings.HasPrefix(input, "/") {
		return "", false
	}
	if strings.ContainsAny(input, "\r\n") {
		return "", false
	}
	if m.textInput.Position() != utf8.RuneCountInString(input) {
		return "", false
	}
	return input, true
}

func (m *Model) refreshSlashSuggestions() {
	oldRows := m.visibleSlashSuggestionChromeRowCount()
	prefix, ok := m.currentSlashSuggestionPrefix()
	if !ok {
		m.slashSuggestions = m.slashSuggestions.Clear()
		m.afterSlashSuggestionChange(oldRows)
		return
	}

	suggestions := m.suggestionsForSlashPrefix(prefix)
	if len(suggestions) == 0 {
		m.slashSuggestions = m.slashSuggestions.Clear()
		m.afterSlashSuggestionChange(oldRows)
		return
	}

	m.slashSuggestions = m.slashSuggestions.Refresh(prefix, suggestions)
	m.afterSlashSuggestionChange(oldRows)
}

func (m *Model) clearSlashSuggestions() {
	if !m.slashSuggestions.Visible() {
		return
	}
	oldRows := m.visibleSlashSuggestionChromeRowCount()
	m.slashSuggestions = m.slashSuggestions.Clear()
	m.afterSlashSuggestionChange(oldRows)
}

func (m *Model) afterSlashSuggestionChange(oldRows int) {
	newRows := m.visibleSlashSuggestionChromeRowCount()
	if oldRows != newRows {
		m.syncComposerLayout()
		return
	}
	m.chromeDirty = true
}

func (m Model) visibleSlashSuggestionChromeRowCount() int {
	return m.slashSuggestions.ChromeRowCount(m.remainingFooterRowsAfterComposerAndAttachments())
}

func (m Model) visibleSlashSuggestionRows() []slash.Suggestion {
	return m.slashSuggestions.VisibleRows(m.remainingFooterRowsAfterComposerAndAttachments())
}

func (m Model) visibleSlashSuggestionRenderRows() []slashsuggestions.RenderRow {
	return m.slashSuggestions.VisibleRenderRows(m.remainingFooterRowsAfterComposerAndAttachments())
}

func (m Model) visibleSlashSuggestionDetailRowCount() int {
	rows := m.visibleSlashSuggestionChromeRowCount() - len(m.visibleSlashSuggestionRows())
	if rows < 0 {
		return 0
	}
	return rows
}

func (m Model) selectedSlashSuggestionDetailText() string {
	return m.slashSuggestions.SelectedDetail()
}

func (m Model) suggestionsForSlashPrefix(prefix string) []slash.Suggestion {
	if suggestions, ok := m.runtimeArgumentSuggestions(prefix); ok {
		return suggestions
	}
	return slash.Suggestions(prefix)
}

func (m Model) runtimeArgumentSuggestions(prefix string) ([]slash.Suggestion, bool) {
	if argPrefix, ok := parseSkillsShowArgument(prefix); ok {
		return m.skillNameArgumentSuggestions(argPrefix), true
	}

	command, argPrefix, ok := parseSingleSlashArgument(prefix)
	if !ok {
		return nil, false
	}
	switch command {
	case "/skills":
		return m.skillsArgumentSuggestions(prefix, argPrefix), true
	case "/provider":
		return providerArgumentSuggestions(m.providerModels.ProviderCandidates(), argPrefix), true
	case "/model":
		provider := strings.TrimSpace(m.configAgent.GetProviderConfigKey())
		if provider == "" {
			provider = strings.TrimSpace(m.configAgent.GetProviderName())
		}
		return modelArgumentSuggestions(m.providerModels.ModelCandidates(provider), argPrefix), true
	default:
		return nil, false
	}
}

func parseSingleSlashArgument(input string) (command, argPrefix string, ok bool) {
	input = strings.TrimLeft(input, " \t")
	idx := strings.IndexAny(input, " \t")
	if idx < 0 {
		return "", "", false
	}
	command = input[:idx]
	argPrefix = strings.TrimLeft(input[idx:], " \t")
	if strings.ContainsAny(argPrefix, " \t") {
		return command, "", false
	}
	return command, argPrefix, true
}

func (m *Model) setInputToSlashSuggestion(suggestion slash.Suggestion, appendArgSpace bool) {
	value := suggestion.CompletionText(appendArgSpace)
	m.textInput.SetValue(value)
	m.textInput.SetCursor(utf8.RuneCountInString(value))
	m.clearSlashSuggestions()
	m.chromeDirty = true
}

func (m *Model) expandSlashSuggestion(suggestion slash.Suggestion) {
	value := suggestion.CompletionText(true)
	if !strings.HasSuffix(value, " ") {
		value += " "
	}
	m.textInput.SetValue(value)
	m.textInput.SetCursor(utf8.RuneCountInString(value))
	m.refreshSlashSuggestions()
	if m.slashSuggestions.Visible() {
		m.slashSuggestions.ActivateSelection()
	}
	m.chromeDirty = true
}

func (m Model) expandSlashSuggestionOnSubmit() (Model, bool) {
	prefix, ok := m.currentSlashSuggestionPrefix()
	if !ok {
		return m, false
	}
	suggestions := m.suggestionsForSlashPrefix(prefix)
	if len(suggestions) != 1 {
		return m, false
	}
	suggestion := suggestions[0]
	if !suggestion.ExpandOnEnter || suggestion.SubmitOnEnter {
		return m, false
	}
	if strings.TrimSpace(prefix) != suggestion.InsertText {
		return m, false
	}
	m.expandSlashSuggestion(suggestion)
	return m, true
}

func (m *Model) setInputToSlashSuggestionSubmission(suggestion slash.Suggestion) {
	value := suggestion.SubmissionText()
	m.textInput.SetValue(value)
	m.textInput.SetCursor(utf8.RuneCountInString(value))
	m.clearSlashSuggestions()
	m.chromeDirty = true
}

func (m Model) handleSlashSuggestionKey(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	result := m.slashSuggestions.HandleKey(msg, m.remainingFooterRowsAfterComposerAndAttachments())
	if !result.Handled {
		return m, nil, false
	}
	m.chromeDirty = true
	switch result.Command {
	case slashsuggestions.KeyCommandClear:
		m.syncComposerLayout()
	case slashsuggestions.KeyCommandCompleteWithSpace:
		m.setInputToSlashSuggestion(result.Suggestion, true)
	case slashsuggestions.KeyCommandExpand:
		m.expandSlashSuggestion(result.Suggestion)
	case slashsuggestions.KeyCommandComplete:
		m.setInputToSlashSuggestion(result.Suggestion, false)
	case slashsuggestions.KeyCommandSubmit:
		m.setInputToSlashSuggestionSubmission(result.Suggestion)
		updated, cmd := m.handleComposerSubmit()
		if typed, ok := updated.(Model); ok {
			m = typed
		}
		return m, cmd, true
	}
	return m, nil, true
}
