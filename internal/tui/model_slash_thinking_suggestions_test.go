package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSlashSuggestions_ShowThinkingArgumentSuggestions(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/thinking ")

	if !m.slashSuggestions.visible() {
		t.Fatal("thinking argument suggestions should be visible")
	}
	rendered := stripANSI(m.chromeCache)
	if !strings.Contains(rendered, "xhigh (max)") {
		t.Fatalf("chromeCache missing xhigh max suggestion:\n%s", rendered)
	}
	if strings.Contains(rendered, "/thinking xhigh") || strings.Contains(rendered, "on · Enable") {
		t.Fatalf("thinking argument suggestions should avoid repeated command/value detail:\n%s", rendered)
	}
	rows := m.renderSlashSuggestionRows()
	if len(rows) == 0 {
		t.Fatal("thinking argument suggestion rows should render")
	}
	if first := stripANSI(rows[0]); !strings.HasPrefix(first, "› on") {
		t.Fatalf("thinking argument suggestions should not reserve an empty category column:\n%s", first)
	}
}

func TestSlashSuggestions_EnterOnThinkingCommandOpensArgumentSuggestions(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if suggestion, ok := m.slashSuggestions.selectedSuggestion(); !ok || suggestion.InsertText != "/thinking" {
		t.Fatalf("selected suggestion = %#v, %v, want /thinking", suggestion, ok)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("Enter on /thinking should open argument suggestions, got cmd %v", cmd)
	}
	if got := m.textInput.Value(); got != "/thinking " {
		t.Fatalf("textInput after Enter = %q, want '/thinking '", got)
	}
	if !m.slashSuggestions.visible() {
		t.Fatal("thinking argument suggestions should be visible after Enter on /thinking")
	}
	if got := len(m.slashSuggestions.suggestions); got != 6 {
		t.Fatalf("thinking argument suggestions len = %d, want 6", got)
	}
	if got := len(agent.handledInputs); got != 0 {
		t.Fatalf("handledInputs length = %d, want 0", got)
	}
}

func TestSlashSuggestions_TabCompletesThinkingArgument(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/thinking x")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("Tab completion should not execute command, got cmd %v", cmd)
	}
	if got := m.textInput.Value(); got != "/thinking xhigh" {
		t.Fatalf("textInput after Tab = %q, want /thinking xhigh", got)
	}
	if m.slashSuggestions.visible() {
		t.Fatal("slash suggestions should close after argument completion")
	}
}

func TestSlashSuggestions_EnterExecutesThinkingArgument(t *testing.T) {
	agent := &stubAgent{
		statusLine:      "ready",
		handledCommands: map[string]bool{"/thinking xhigh": true},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/thinking x")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	requireAgentDoneCmd(t, cmd)
	if got := len(agent.handledInputs); got != 1 {
		t.Fatalf("handledInputs length = %d, want 1", got)
	}
	if got := agent.handledInputs[0]; got != "/thinking xhigh" {
		t.Fatalf("handledInputs[0] = %q, want /thinking xhigh", got)
	}
	if m.textInput.Value() != "" {
		t.Fatalf("textInput after command = %q, want empty", m.textInput.Value())
	}
}

func TestSlashSuggestions_EnterExecutesDefaultThinkingArgument(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "canonical", input: "/thinking ", want: "/thinking on"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &stubAgent{
				statusLine:      "ready",
				handledCommands: map[string]bool{tt.want: true},
			}
			m := newModelWithViewport(agent)
			m = sendComposerRunes(m, tt.input)

			if !m.slashSuggestions.visible() {
				t.Fatal("thinking argument suggestions should be visible before Enter")
			}

			updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = updated.(Model)

			requireAgentDoneCmd(t, cmd)
			if got := len(agent.handledInputs); got != 1 {
				t.Fatalf("handledInputs length = %d, want 1", got)
			}
			if got := agent.handledInputs[0]; got != tt.want {
				t.Fatalf("handledInputs[0] = %q, want %q", got, tt.want)
			}
			if m.textInput.Value() != "" {
				t.Fatalf("textInput after command = %q, want empty", m.textInput.Value())
			}
			if m.slashSuggestions.visible() {
				t.Fatal("slash suggestions should close after thinking argument execution")
			}
		})
	}
}

func TestSlashSuggestions_EnterOnExpandedThinkingArgumentExecutesDefault(t *testing.T) {
	agent := &stubAgent{
		statusLine:      "ready",
		handledCommands: map[string]bool{"/thinking on": true},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if suggestion, ok := m.slashSuggestions.selectedSuggestion(); !ok || suggestion.InsertText != "/thinking" {
		t.Fatalf("selected suggestion = %#v, %v, want /thinking", suggestion, ok)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("Enter on /thinking should open argument suggestions, got cmd %v", cmd)
	}
	if got := m.textInput.Value(); got != "/thinking " {
		t.Fatalf("textInput after expanding /thinking = %q, want '/thinking '", got)
	}
	if !m.slashSuggestions.visible() {
		t.Fatal("thinking argument suggestions should be visible after expanding /thinking")
	}

	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	requireAgentDoneCmd(t, cmd)
	if got := len(agent.handledInputs); got != 1 {
		t.Fatalf("handledInputs length = %d, want 1", got)
	}
	if got := agent.handledInputs[0]; got != "/thinking on" {
		t.Fatalf("handledInputs[0] = %q, want /thinking on", got)
	}
	if m.textInput.Value() != "" {
		t.Fatalf("textInput after command = %q, want empty", m.textInput.Value())
	}
	if m.slashSuggestions.visible() {
		t.Fatal("slash suggestions should close after thinking argument execution")
	}
}

func TestSlashSuggestions_DownEnterExecutesThinkingArgument(t *testing.T) {
	agent := &stubAgent{
		statusLine:      "ready",
		handledCommands: map[string]bool{"/thinking off": true},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/thinking ")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	if got := len(agent.handledInputs); got != 0 {
		t.Fatalf("handledInputs after Down = %d, want 0", got)
	}
	if !m.slashSuggestions.selectionActive {
		t.Fatal("Down should activate thinking argument selection")
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	requireAgentDoneCmd(t, cmd)
	if got := len(agent.handledInputs); got != 1 {
		t.Fatalf("handledInputs length = %d, want 1", got)
	}
	if got := agent.handledInputs[0]; got != "/thinking off" {
		t.Fatalf("handledInputs[0] = %q, want /thinking off", got)
	}
	if m.textInput.Value() != "" {
		t.Fatalf("textInput after command = %q, want empty", m.textInput.Value())
	}
}

func TestSlashSuggestions_EnterOnTypedThinkingOpensArgumentSuggestions(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/thinking")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("Enter on typed /thinking should open argument suggestions, got cmd %v", cmd)
	}
	if got := len(agent.handledInputs); got != 0 {
		t.Fatalf("handledInputs length = %d, want 0", got)
	}
	if got := m.textInput.Value(); got != "/thinking " {
		t.Fatalf("textInput after Enter = %q, want '/thinking '", got)
	}
	if !m.slashSuggestions.visible() {
		t.Fatal("thinking argument suggestions should be visible after Enter on typed /thinking")
	}
	suggestion, ok := m.slashSuggestions.selectedSuggestion()
	if !ok || suggestion.InsertText != "/thinking on" {
		t.Fatalf("selected suggestion = %#v, %v, want /thinking on", suggestion, ok)
	}
}

func TestSlashSuggestions_ThinkingAliasArgumentsAreRemoved(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/think ")

	if m.slashSuggestions.visible() {
		t.Fatalf("removed /think alias should not show argument suggestions: %#v", m.slashSuggestions.suggestions)
	}
}
