package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newSurfaceDispatchTestAgent(t *testing.T) (*Agent, *bytes.Buffer) {
	t.Helper()
	withTempWorkdir(t)
	t.Setenv("HOME", t.TempDir())

	cfg := newProjectMapDisabledConfig()
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	out := &bytes.Buffer{}
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(""), out, out)

	return &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-test",
		CurrentProvider: &MockProvider{name: "openai"},
		Runtime:         runtime,
	}, out
}

func TestResolveConfigCommandSurfacePolicy(t *testing.T) {
	tests := []struct {
		name    string
		surface commandcatalog.CommandSurface
		args    []string
		want    configCommandSurfacePolicy
	}{
		{
			name:    "classic bare dispatches",
			surface: commandcatalog.CommandSurfaceClassic,
			want:    configCommandSurfacePolicy{dispatch: true},
		},
		{
			name:    "TUI show dispatches",
			surface: commandcatalog.CommandSurfaceTUI,
			args:    []string{"show"},
			want:    configCommandSurfacePolicy{dispatch: true},
		},
		{
			name:    "TUI model with value dispatches",
			surface: commandcatalog.CommandSurfaceTUI,
			args:    []string{"model", "gpt-new"},
			want:    configCommandSurfacePolicy{dispatch: true},
		},
		{
			name:    "TUI bare warns",
			surface: commandcatalog.CommandSurfaceTUI,
			want:    configCommandSurfacePolicy{warningCommand: "/config"},
		},
		{
			name:    "TUI unknown subcommand warns",
			surface: commandcatalog.CommandSurfaceTUI,
			args:    []string{"foo"},
			want:    configCommandSurfacePolicy{warningCommand: "/config foo"},
		},
		{
			name:    "TUI incomplete model warns",
			surface: commandcatalog.CommandSurfaceTUI,
			args:    []string{"model"},
			want:    configCommandSurfacePolicy{warningCommand: "/config model"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveConfigCommandSurfacePolicy(tt.args, tt.surface); got != tt.want {
				t.Fatalf("resolveConfigCommandSurfacePolicy(%v, %s) = %#v, want %#v", tt.args, tt.surface, got, tt.want)
			}
		})
	}
}

func TestSpecialCommandRegistryDoesNotRegisterTUILocalOwners(t *testing.T) {
	agent, _ := newSurfaceDispatchTestAgent(t)

	for _, surface := range []commandcatalog.CommandSurface{
		commandcatalog.CommandSurfaceClassic,
		commandcatalog.CommandSurfaceTUI,
	} {
		t.Run(string(surface), func(t *testing.T) {
			registry := specialCommandRegistry(agent, surface)
			for _, cmd := range commandcatalog.Commands {
				if cmd.EffectiveOwner() != commandcatalog.CommandOwnerTUIRouter {
					continue
				}
				names := append([]string{cmd.Name}, cmd.Aliases...)
				for _, name := range names {
					if _, ok := registry[name]; ok {
						t.Fatalf("%s command %s should be owned by TUI router, not agent registry", surface, name)
					}
				}
			}
		})
	}
}

func TestHandleSpecialCommandForSurface_Matrix(t *testing.T) {
	disableColors(t)

	tests := []struct {
		name        string
		surface     commandcatalog.CommandSurface
		input       string
		wantHandled bool
		wantText    string
		rejectText  string
	}{
		{
			name:        "classic bare review warns",
			surface:     commandcatalog.CommandSurfaceClassic,
			input:       "/review",
			wantHandled: true,
			wantText:    "/review is available in TUI mode only",
		},
		{
			name:        "classic non-bare review falls through",
			surface:     commandcatalog.CommandSurfaceClassic,
			input:       "/review staged",
			wantHandled: false,
		},
		{
			name:        "TUI bare review is local screen owned",
			surface:     commandcatalog.CommandSurfaceTUI,
			input:       "/review",
			wantHandled: false,
		},
		{
			name:        "TUI non-bare review falls through",
			surface:     commandcatalog.CommandSurfaceTUI,
			input:       "/review staged",
			wantHandled: false,
		},
		{
			name:        "classic bare project warns",
			surface:     commandcatalog.CommandSurfaceClassic,
			input:       "/project",
			wantHandled: true,
			wantText:    "/project is available in TUI mode only",
		},
		{
			name:        "classic non-bare project falls through",
			surface:     commandcatalog.CommandSurfaceClassic,
			input:       "/project rules",
			wantHandled: false,
		},
		{
			name:        "TUI bare project is local screen owned",
			surface:     commandcatalog.CommandSurfaceTUI,
			input:       "/project",
			wantHandled: false,
		},
		{
			name:        "TUI non-bare project falls through",
			surface:     commandcatalog.CommandSurfaceTUI,
			input:       "/project rules",
			wantHandled: false,
		},
		{
			name:        "TUI bare config direct dispatch is rejected",
			surface:     commandcatalog.CommandSurfaceTUI,
			input:       "/config",
			wantHandled: true,
			wantText:    "Use bare /config, /config show, or /config model <name>.",
		},
		{
			name:        "TUI unsupported config subcommand is rejected",
			surface:     commandcatalog.CommandSurfaceTUI,
			input:       "/config foo",
			wantHandled: true,
			wantText:    "/config foo is not available in TUI mode",
		},
		{
			name:        "TUI config show remains dispatchable",
			surface:     commandcatalog.CommandSurfaceTUI,
			input:       "/config show",
			wantHandled: true,
			wantText:    "default_provider",
			rejectText:  "not available in TUI mode",
		},
		{
			name:        "TUI bare lsp warns classic only",
			surface:     commandcatalog.CommandSurfaceTUI,
			input:       "/lsp",
			wantHandled: true,
			wantText:    "/lsp is available in classic mode only",
		},
		{
			name:        "TUI non-bare lsp falls through",
			surface:     commandcatalog.CommandSurfaceTUI,
			input:       "/lsp status",
			wantHandled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, out := newSurfaceDispatchTestAgent(t)
			got := handleSpecialCommandForSurface(tt.input, agent, tt.surface)
			if got != tt.wantHandled {
				t.Fatalf("handleSpecialCommandForSurface(%q, %s) = %v, want %v", tt.input, tt.surface, got, tt.wantHandled)
			}
			output := out.String()
			if tt.wantText != "" && !strings.Contains(output, tt.wantText) {
				t.Fatalf("output missing %q:\n%s", tt.wantText, output)
			}
			if tt.rejectText != "" && strings.Contains(output, tt.rejectText) {
				t.Fatalf("output should not contain %q:\n%s", tt.rejectText, output)
			}
		})
	}
}
