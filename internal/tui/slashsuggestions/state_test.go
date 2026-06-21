package slashsuggestions

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/slash"
)

func TestHandleKeyRawEnterSubmitsSelectedSuggestion(t *testing.T) {
	for _, tt := range rawEnterKeyCases() {
		t.Run(tt.name, func(t *testing.T) {
			state := State{}.Refresh("/", []slash.Suggestion{
				{Label: "/review", InsertText: "/review", SubmitOnEnter: true},
			})

			result := state.HandleKey(tt.msg, 8)

			if result.Command != KeyCommandSubmit || !result.Handled {
				t.Fatalf("HandleKey() = %#v, want submit handled", result)
			}
			if result.Suggestion.InsertText != "/review" {
				t.Fatalf("Suggestion = %#v, want /review", result.Suggestion)
			}
		})
	}
}

func TestHandleKeyRawEnterExpandsSelectedSuggestion(t *testing.T) {
	for _, tt := range rawEnterKeyCases() {
		t.Run(tt.name, func(t *testing.T) {
			state := State{}.Refresh("/", []slash.Suggestion{
				{Label: "/skills", InsertText: "/skills", ExpandOnEnter: true},
			})

			result := state.HandleKey(tt.msg, 8)

			if result.Command != KeyCommandExpand || !result.Handled {
				t.Fatalf("HandleKey() = %#v, want expand handled", result)
			}
			if result.Suggestion.InsertText != "/skills" {
				t.Fatalf("Suggestion = %#v, want /skills", result.Suggestion)
			}
		})
	}
}

func TestHandleKeyRawEnterCompletesActiveSuggestion(t *testing.T) {
	for _, tt := range rawEnterKeyCases() {
		t.Run(tt.name, func(t *testing.T) {
			state := State{}.Refresh("/", []slash.Suggestion{
				{Label: "bug-investigation", InsertText: "/skill bug-investigation", CompleteOnEnter: true},
			})
			state.ActivateSelection()

			result := state.HandleKey(tt.msg, 8)

			if result.Command != KeyCommandComplete || !result.Handled {
				t.Fatalf("HandleKey() = %#v, want complete handled", result)
			}
			if result.Suggestion.InsertText != "/skill bug-investigation" {
				t.Fatalf("Suggestion = %#v, want skill completion", result.Suggestion)
			}
		})
	}
}

func rawEnterKeyCases() []struct {
	name string
	msg  tea.KeyMsg
} {
	return []struct {
		name string
		msg  tea.KeyMsg
	}{
		{name: "raw carriage return", msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\r'}}},
		{name: "raw line feed", msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\n'}}},
		{name: "bubbletea line feed", msg: tea.KeyMsg{Type: tea.KeyCtrlJ}},
	}
}
