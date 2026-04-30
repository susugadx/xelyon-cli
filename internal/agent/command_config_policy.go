package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
)

type configCommandSurfacePolicy struct {
	dispatch       bool
	warningCommand string
}

func handleConfigCommandForSurface(agent *Agent, args []string, commandSurface commandcatalog.CommandSurface) bool {
	policy := resolveConfigCommandSurfacePolicy(args, commandSurface)
	if policy.dispatch {
		return handleConfigCommand(agent, args)
	}

	_, _ = yellow.Fprintf(agent.output(), "⚠️  %s is not available in TUI mode.\n", policy.warningCommand)
	_, _ = yellow.Fprintln(agent.output(), "   Use bare /config, /config show, or /config model <name>.")
	return true
}

func resolveConfigCommandSurfacePolicy(args []string, commandSurface commandcatalog.CommandSurface) configCommandSurfacePolicy {
	if commandSurface != commandcatalog.CommandSurfaceTUI {
		return configCommandSurfacePolicy{dispatch: true}
	}
	if isNonInteractiveConfigSubcommand(args) {
		return configCommandSurfacePolicy{dispatch: true}
	}
	return configCommandSurfacePolicy{warningCommand: formatConfigCommand(args)}
}

func formatConfigCommand(args []string) string {
	if len(args) == 0 {
		return "/config"
	}
	return "/config " + strings.Join(args, " ")
}
