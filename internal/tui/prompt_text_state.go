package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type promptTextState struct {
	input          textinput.Model
	value          string
	responseAction ui.PromptAction
	defaultValue   string
}

func newPromptTextState(req ui.PromptRequest) promptTextState {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = req.Placeholder
	input.CharLimit = 0
	input.Width = 80

	return promptTextState{
		input:        input,
		defaultValue: req.DefaultValue,
	}
}

func (s *promptTextState) focus() {
	s.input.Focus()
}

func (s *promptTextState) beginComment(placeholder string) {
	s.responseAction = ui.PromptActionComment
	s.setValue("")
	if placeholder == "" {
		placeholder = "Type feedback. Use image:/path to attach an image."
	}
	s.input.Placeholder = placeholder
	s.input.Focus()
}

func (s promptTextState) response() ui.PromptResponse {
	text := s.value
	if text == "" && s.defaultValue != "" {
		text = s.defaultValue
	}

	resp := ui.PromptResponse{Text: text}
	if s.responseAction != "" {
		resp.Action = s.responseAction
	}
	return resp
}

func (s promptTextState) viewLine() string {
	return termtext.SanitizeSingleLineANSI(termtext.StripANSI(s.input.View()))
}

func (s *promptTextState) update(msg tea.KeyMsg) tea.Cmd {
	if msg.Paste {
		s.appendValue(string(msg.Runes))
		return nil
	}
	if promptTextHasLineBreak(s.value) {
		s.updateMultilineValue(msg)
		return nil
	}

	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	s.value = s.input.Value()
	return cmd
}

func (s *promptTextState) appendValue(text string) {
	s.setValue(s.value + normalizePromptTextValue(text))
}

func (s *promptTextState) setValue(text string) {
	s.value = normalizePromptTextValue(text)
	s.input.SetValue(promptTextDisplayValue(s.value))
}

func (s *promptTextState) updateMultilineValue(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyRunes:
		s.appendValue(string(msg.Runes))
	case tea.KeyBackspace, tea.KeyCtrlH:
		s.trimLastRune()
	}
}

func (s *promptTextState) trimLastRune() {
	runes := []rune(s.value)
	if len(runes) == 0 {
		return
	}
	s.setValue(string(runes[:len(runes)-1]))
}

func normalizePromptTextValue(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n")
}

func promptTextDisplayValue(text string) string {
	text = normalizePromptTextValue(text)
	return strings.ReplaceAll(text, "\n", "\\n")
}

func promptTextHasLineBreak(text string) bool {
	return strings.ContainsAny(text, "\r\n")
}
