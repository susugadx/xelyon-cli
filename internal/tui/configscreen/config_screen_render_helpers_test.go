package configscreen

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestConfigStatusHint(t *testing.T) {
	tests := []struct {
		name string
		cs   *Screen
		want string
	}{
		{
			name: "filter mode",
			cs:   &Screen{filterMode: true},
			want: "Enter:apply  Esc:cancel",
		},
		{
			name: "slice editor idle",
			cs:   &Screen{editMode: editSlice},
			want: "a:add  d:delete  Enter:edit  Esc:done",
		},
		{
			name: "slice editor editing",
			cs:   &Screen{editMode: editSlice, editSliceEditing: true},
			want: "Enter:confirm  Esc:cancel",
		},
		{
			name: "struct entry input",
			cs:   &Screen{editMode: editStructMap, editEntryActive: true, editEntryFieldEdit: "input"},
			want: "Enter:confirm  Esc:cancel",
		},
		{
			name: "default browse",
			cs:   &Screen{},
			want: "j/k:move  h/l:pane  Enter:edit  Space:toggle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := configStatusHint(tt.cs); got != tt.want {
				t.Fatalf("configStatusHint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatConfigValue(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		want string
	}{
		{name: "nil", v: nil, want: "(nil)"},
		{name: "empty string", v: "", want: "(empty)"},
		{name: "bool true", v: true, want: "true"},
		{name: "slice", v: []string{"a", "b"}, want: "2 items"},
		{name: "provider map", v: map[string]config.ProviderModelConfig{"openai": {}}, want: "1 entries"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatConfigValue(tt.v, config.FieldTypeString); got != tt.want {
				t.Fatalf("formatConfigValue() = %q, want %q", got, tt.want)
			}
		})
	}
}
