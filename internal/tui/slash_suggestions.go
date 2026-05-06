package tui

import (
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/tui/slash"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

const maxSlashSuggestionRows = 8

type slashSuggestionState struct {
	prefix          string
	suggestions     []slash.Suggestion
	selected        int
	selectionActive bool
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
	oldRows := len(m.visibleSlashSuggestionRows())
	prefix, ok := m.currentSlashSuggestionPrefix()
	if !ok {
		m.slashSuggestions = slashSuggestionState{}
		m.afterSlashSuggestionChange(oldRows)
		return
	}

	suggestions := slash.Suggestions(prefix)
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
	oldRows := len(m.visibleSlashSuggestionRows())
	m.slashSuggestions = slashSuggestionState{}
	m.afterSlashSuggestionChange(oldRows)
}

func (m *Model) afterSlashSuggestionChange(oldRows int) {
	newRows := len(m.visibleSlashSuggestionRows())
	if oldRows != newRows {
		m.syncComposerLayout()
		return
	}
	m.chromeDirty = true
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

func (m Model) maxVisibleSlashSuggestionRows() int {
	available := m.remainingFooterRowsAfterComposerAndAttachments()
	if available <= 0 {
		return 0
	}
	if available > maxSlashSuggestionRows {
		return maxSlashSuggestionRows
	}
	return available
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

func paddedPlainText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	text = termtext.TruncateWithANSI(termtext.SanitizeSingleLineANSI(text), width)
	padding := width - lipgloss.Width(text)
	if padding > 0 {
		text += strings.Repeat(" ", padding)
	}
	return text
}
