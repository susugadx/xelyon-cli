package keyinput

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestIsEnterKey(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyMsg
		want bool
	}{
		{
			name: "key enter",
			msg:  tea.KeyMsg{Type: tea.KeyEnter},
			want: true,
		},
		{
			name: "key ctrl j line feed",
			msg:  tea.KeyMsg{Type: tea.KeyCtrlJ},
			want: true,
		},
		{
			name: "string enter fallback",
			msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("enter")},
			want: true,
		},
		{
			name: "raw carriage return fallback",
			msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\r'}},
			want: true,
		},
		{
			name: "raw line feed fallback",
			msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\n'}},
			want: true,
		},
		{
			name: "non enter rune",
			msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")},
			want: false,
		},
		{
			name: "literal ctrl j rune",
			msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ctrl+j")},
			want: false,
		},
		{
			name: "escape",
			msg:  tea.KeyMsg{Type: tea.KeyEsc},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEnterKey(tt.msg); got != tt.want {
				t.Fatalf("IsEnterKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
