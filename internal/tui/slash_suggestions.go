package tui

import (
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/providerpicker"
	"github.com/susugadx/xelyon-cli/internal/tui/slash"
)

const maxSlashSuggestionRows = 8

type slashSuggestionState struct {
	prefix          string
	suggestions     []slash.Suggestion
	selected        int
	selectionActive bool
}

type slashSuggestionRenderRow struct {
	Category     string
	CommandLabel string
	Description  string
	Selected     bool
}

func (s slashSuggestionState) visible() bool {
	return len(s.suggestions) > 0
}

func (s slashSuggestionState) selectedSuggestion() (slash.Suggestion, bool) {
	if !s.visible() || s.selected < 0 || s.selected >= len(s.suggestions) {
		return slash.Suggestion{}, false
	}
	return s.suggestions[s.selected], true
}

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
		m.slashSuggestions = slashSuggestionState{}
		m.afterSlashSuggestionChange(oldRows)
		return
	}

	suggestions := m.suggestionsForSlashPrefix(prefix)
	if len(suggestions) == 0 {
		m.slashSuggestions = slashSuggestionState{}
		m.afterSlashSuggestionChange(oldRows)
		return
	}

	selected := m.slashSuggestions.selected
	selectionActive := m.slashSuggestions.selectionActive
	if prefix != m.slashSuggestions.prefix {
		selected = 0
		selectionActive = false
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= len(suggestions) {
		selected = len(suggestions) - 1
	}
	m.slashSuggestions = slashSuggestionState{
		prefix:          prefix,
		suggestions:     suggestions,
		selected:        selected,
		selectionActive: selectionActive,
	}
	m.afterSlashSuggestionChange(oldRows)
}

func (m *Model) clearSlashSuggestions() {
	if !m.slashSuggestions.visible() {
		return
	}
	oldRows := m.visibleSlashSuggestionChromeRowCount()
	m.slashSuggestions = slashSuggestionState{}
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
	return len(m.visibleSlashSuggestionRows()) + m.visibleSlashSuggestionDetailRowCount()
}

func (m Model) visibleSlashSuggestionRows() []slash.Suggestion {
	if !m.slashSuggestions.visible() {
		return nil
	}
	limit := m.maxVisibleSlashSuggestionRows()
	if limit <= 0 {
		return nil
	}
	start := m.slashSuggestionWindowStart()
	end := start + limit
	if end > len(m.slashSuggestions.suggestions) {
		end = len(m.slashSuggestions.suggestions)
	}
	return m.slashSuggestions.suggestions[start:end]
}

func (m Model) visibleSlashSuggestionRenderRows() []slashSuggestionRenderRow {
	rows := m.visibleSlashSuggestionRows()
	if len(rows) == 0 {
		return nil
	}
	start := m.slashSuggestionWindowStart()
	out := make([]slashSuggestionRenderRow, 0, len(rows))
	for i, suggestion := range rows {
		out = append(out, newSlashSuggestionRenderRow(suggestion, start+i == m.slashSuggestions.selected))
	}
	return out
}

func newSlashSuggestionRenderRow(suggestion slash.Suggestion, selected bool) slashSuggestionRenderRow {
	return slashSuggestionRenderRow{
		Category:     suggestion.CategoryDisplayLabel(),
		CommandLabel: suggestion.Label,
		Description:  suggestion.Description,
		Selected:     selected,
	}
}

func (m Model) maxVisibleSlashSuggestionRows() int {
	available := m.remainingFooterRowsAfterComposerAndAttachments()
	if m.selectedSlashSuggestionDetailText() != "" && available > 1 {
		available--
	}
	if available <= 0 {
		return 0
	}
	if available > maxSlashSuggestionRows {
		return maxSlashSuggestionRows
	}
	return available
}

func (m Model) visibleSlashSuggestionDetailRowCount() int {
	if m.selectedSlashSuggestionDetailText() == "" ||
		m.remainingFooterRowsAfterComposerAndAttachments() <= 1 ||
		len(m.visibleSlashSuggestionRows()) == 0 {
		return 0
	}
	return 1
}

func (m Model) selectedSlashSuggestionDetailText() string {
	suggestion, ok := m.slashSuggestions.selectedSuggestion()
	if !ok {
		return ""
	}
	return slashSuggestionDetailText(suggestion)
}

func slashSuggestionDetailText(suggestion slash.Suggestion) string {
	detail := strings.TrimSpace(suggestion.Detail)
	if detail == "" {
		detail = strings.TrimSpace(suggestion.Description)
	}
	argHint := strings.TrimSpace(suggestion.ArgHint)
	if argHint != "" && detail != "" {
		return argHint + " · " + detail
	}
	if argHint != "" {
		return argHint
	}
	return detail
}

func (m Model) suggestionsForSlashPrefix(prefix string) []slash.Suggestion {
	if suggestions, ok := m.runtimeArgumentSuggestions(prefix); ok {
		return suggestions
	}
	return slash.Suggestions(prefix)
}

func (m Model) runtimeArgumentSuggestions(prefix string) ([]slash.Suggestion, bool) {
	command, argPrefix, ok := parseSingleSlashArgument(prefix)
	if !ok {
		return nil, false
	}
	switch command {
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

func providerArgumentSuggestions(candidates []providerpicker.ProviderCandidate, argPrefix string) []slash.Suggestion {
	argPrefix = strings.ToLower(strings.TrimSpace(argPrefix))
	submitOnEnter := argPrefix != ""
	suggestions := make([]slash.Suggestion, 0, len(candidates))
	for _, candidate := range candidates {
		if !strings.HasPrefix(strings.ToLower(candidate.Key), argPrefix) {
			continue
		}
		insertText := "/provider " + candidate.Key
		suggestions = append(suggestions, slash.Suggestion{
			Label:         insertText,
			InsertText:    insertText,
			Description:   providerCandidateDescription(candidate),
			Category:      commandcatalog.CommandCategoryModel,
			CategoryLabel: "provider",
			ArgHint:       candidate.Key,
			Detail:        "Switch to " + candidate.Label,
			SubmitOnEnter: submitOnEnter,
		})
	}
	return suggestions
}

func modelArgumentSuggestions(candidates []providerpicker.ModelCandidate, argPrefix string) []slash.Suggestion {
	argPrefix = strings.ToLower(strings.TrimSpace(argPrefix))
	submitOnEnter := argPrefix != ""
	suggestions := make([]slash.Suggestion, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Custom || !strings.HasPrefix(strings.ToLower(candidate.Name), argPrefix) {
			continue
		}
		insertText := "/model " + candidate.Name
		suggestions = append(suggestions, slash.Suggestion{
			Label:         insertText,
			InsertText:    insertText,
			Description:   modelCandidateDescription(candidate),
			Category:      commandcatalog.CommandCategoryModel,
			CategoryLabel: "model",
			ArgHint:       candidate.Name,
			Detail:        "Switch current provider model to " + candidate.Name,
			SubmitOnEnter: submitOnEnter,
		})
	}
	return suggestions
}

func providerCandidateDescription(candidate providerpicker.ProviderCandidate) string {
	var parts []string
	if candidate.Label != "" && candidate.Label != candidate.Key {
		parts = append(parts, candidate.Label)
	}
	if candidate.Current {
		parts = append(parts, "current")
	}
	if candidate.CredentialStatus != "" {
		parts = append(parts, string(candidate.CredentialStatus))
	}
	return strings.Join(parts, " · ")
}

func modelCandidateDescription(candidate providerpicker.ModelCandidate) string {
	var parts []string
	if candidate.Current {
		parts = append(parts, "current")
	}
	if candidate.Default {
		parts = append(parts, "default")
	}
	return strings.Join(parts, " · ")
}

func (m Model) slashSuggestionWindowStart() int {
	if !m.slashSuggestions.visible() {
		return 0
	}
	limit := m.maxVisibleSlashSuggestionRows()
	if limit <= 0 || m.slashSuggestions.selected < limit {
		return 0
	}
	return m.slashSuggestions.selected - limit + 1
}

func (m *Model) moveSlashSuggestion(delta int) {
	if !m.slashSuggestions.visible() {
		return
	}
	count := len(m.slashSuggestions.suggestions)
	m.slashSuggestions.selected = (m.slashSuggestions.selected + delta + count) % count
	m.slashSuggestions.selectionActive = true
	m.chromeDirty = true
}

func (m *Model) activateSlashSuggestionSelection() {
	if !m.slashSuggestions.visible() {
		return
	}
	m.slashSuggestions.selectionActive = true
	m.chromeDirty = true
}

func (m *Model) setInputToSlashSuggestion(suggestion slash.Suggestion, appendArgSpace bool) {
	value := suggestion.CompletionText(appendArgSpace)
	m.textInput.SetValue(value)
	m.textInput.SetCursor(utf8.RuneCountInString(value))
	m.clearSlashSuggestions()
	m.chromeDirty = true
}

func (m *Model) setInputToSlashSuggestionSubmission(suggestion slash.Suggestion) {
	value := suggestion.SubmissionText()
	m.textInput.SetValue(value)
	m.textInput.SetCursor(utf8.RuneCountInString(value))
	m.clearSlashSuggestions()
	m.chromeDirty = true
}

func (m Model) handleSlashSuggestionKey(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if len(m.visibleSlashSuggestionRows()) == 0 {
		return m, nil, false
	}

	switch {
	case msg.Type == tea.KeyEsc:
		m.clearSlashSuggestions()
		return m, nil, true
	case msg.Type == tea.KeyUp || msg.Type == tea.KeyShiftTab || msg.String() == "ctrl+p":
		m.moveSlashSuggestion(-1)
		return m, nil, true
	case msg.Type == tea.KeyDown || msg.String() == "ctrl+n":
		m.moveSlashSuggestion(1)
		return m, nil, true
	case msg.Type == tea.KeyTab:
		if suggestion, ok := m.slashSuggestions.selectedSuggestion(); ok {
			m.setInputToSlashSuggestion(suggestion, true)
		}
		return m, nil, true
	case isEnterKey(msg):
		if suggestion, ok := m.slashSuggestions.selectedSuggestion(); ok {
			if !suggestion.SubmitOnEnter && !m.slashSuggestions.selectionActive {
				m.activateSlashSuggestionSelection()
				return m, nil, true
			}
			m.setInputToSlashSuggestionSubmission(suggestion)
			updated, cmd := m.handleComposerSubmit()
			if typed, ok := updated.(Model); ok {
				m = typed
			}
			return m, cmd, true
		}
	}

	return m, nil, false
}
