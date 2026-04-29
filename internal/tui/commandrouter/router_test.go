package commandrouter

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tui/slash"
)

func TestRoute_TUILocalCommands(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		resolve slash.AliasResolver
		ctx     Context
		want    Action
	}{
		{
			name:  "copy selection",
			input: "/copy",
			ctx:   Context{HasMouseSelection: true},
			want:  ActionCopyMouseSelection,
		},
		{
			name:  "copy without selection delegates",
			input: "/copy",
			want:  ActionDispatchAgent,
		},
		{
			name:  "quit alias",
			input: "/q",
			resolve: func(name string) string {
				if name == "/q" {
					return "/quit"
				}
				return name
			},
			want: ActionQuit,
		},
		{
			name:  "config alias bare",
			input: "/c",
			resolve: func(name string) string {
				if name == "/c" {
					return "/config"
				}
				return name
			},
			want: ActionOpenConfig,
		},
		{
			name:  "config with args delegates",
			input: "/config show",
			want:  ActionDispatchAgent,
		},
		{
			name:  "review bare",
			input: "/review",
			want:  ActionOpenReview,
		},
		{
			name:  "review with args delegates",
			input: "/review staged",
			want:  ActionDispatchAgent,
		},
		{
			name:  "project bare",
			input: "/project",
			want:  ActionOpenProject,
		},
		{
			name:  "project with args delegates",
			input: "/project rules",
			want:  ActionDispatchAgent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := slash.NewCommand(tt.input, tt.input, tt.resolve)
			if got := Route(cmd, tt.ctx); got != tt.want {
				t.Fatalf("Route() = %v, want %v", got, tt.want)
			}
		})
	}
}
