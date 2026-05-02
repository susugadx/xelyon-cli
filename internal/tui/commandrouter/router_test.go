package commandrouter

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
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
		{
			name:  "attach with args is local",
			input: "/attach ./notes.txt",
			want:  ActionManageAttachments,
		},
		{
			name:  "detach with args is local",
			input: "/detach 2",
			want:  ActionManageAttachments,
		},
		{
			name:  "detach-all bare is local",
			input: "/detach-all",
			want:  ActionManageAttachments,
		},
		{
			name:  "detach-all with args still local",
			input: "/detach-all now",
			want:  ActionManageAttachments,
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

func TestRoute_CatalogTUILocalOwnerMatrix(t *testing.T) {
	for _, cmdInfo := range commandcatalog.Commands {
		if cmdInfo.EffectiveOwner() != commandcatalog.CommandOwnerTUIRouter {
			continue
		}

		t.Run(cmdInfo.Name, func(t *testing.T) {
			ctx := Context{}
			if cmdInfo.AcceptsTUILocalContext(commandcatalog.TUILocalContext{HasMouseSelection: true}) &&
				!cmdInfo.AcceptsTUILocalContext(commandcatalog.TUILocalContext{HasMouseSelection: false}) {
				ctx.HasMouseSelection = true
			}

			bare := slash.NewCommand(cmdInfo.Name, cmdInfo.Name, nil)
			if got := Route(bare, ctx); got == ActionDispatchAgent {
				t.Fatalf("Route(%q) = ActionDispatchAgent, want TUI-local action", cmdInfo.Name)
			}

			withArgsInput := cmdInfo.Name + " extra"
			withArgs := slash.NewCommand(withArgsInput, withArgsInput, nil)
			gotWithArgs := Route(withArgs, ctx)
			if cmdInfo.AcceptsTUILocalArgs(withArgs.Args) {
				if gotWithArgs == ActionDispatchAgent {
					t.Fatalf("Route(%q) = ActionDispatchAgent, want TUI-local action", withArgsInput)
				}
			} else {
				if gotWithArgs != ActionDispatchAgent {
					t.Fatalf("Route(%q) = %v, want ActionDispatchAgent", withArgsInput, gotWithArgs)
				}
			}
		})
	}
}

func TestKnownLocalAction_CoversAllCatalogLocalActions(t *testing.T) {
	actionsUsedByCatalog := make(map[commandcatalog.TUILocalAction]struct{})
	for _, cmdInfo := range commandcatalog.Commands {
		action := cmdInfo.EffectiveTUILocalAction()
		if action == commandcatalog.TUILocalActionNone {
			continue
		}
		actionsUsedByCatalog[action] = struct{}{}
		if !isKnownLocalAction(Action(action)) {
			t.Fatalf("isKnownLocalAction is missing mapping for %q (command: %s)", action, cmdInfo.Name)
		}
	}

	for _, action := range []commandcatalog.TUILocalAction{
		commandcatalog.TUILocalActionCopyMouseSelection,
		commandcatalog.TUILocalActionManageAttachments,
		commandcatalog.TUILocalActionQuit,
		commandcatalog.TUILocalActionOpenConfig,
		commandcatalog.TUILocalActionOpenReview,
		commandcatalog.TUILocalActionOpenProject,
	} {
		if _, ok := actionsUsedByCatalog[action]; !ok {
			t.Fatalf("catalogActionToRouterAction has stale mapping %q (no command uses it)", action)
		}
	}
}
