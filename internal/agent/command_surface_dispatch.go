package agent

import (
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/commandruntime"
)

// handleSpecialCommand は classic surface の特殊コマンドを処理する。
func handleSpecialCommand(input string, agent *Agent) bool {
	return handleSpecialCommandForSurface(input, agent, commandcatalog.CommandSurfaceClassic)
}

func handleSpecialCommandForSurface(input string, agent *Agent, commandSurface commandcatalog.CommandSurface) bool {
	invocation, ok := commandruntime.Parse(input)
	if !ok {
		return false
	}
	if cmdInfo, known := commandcatalog.Find(invocation.Command); known && !cmdInfo.SupportsSurface(commandSurface) {
		return handleUnsupportedCommandSurface(invocation, cmdInfo, agent, commandSurface)
	}
	handler, ok := specialCommandRegistry(agent, commandSurface)[invocation.Command]
	if !ok {
		return false
	}
	return handler(invocation.Args)
}

func handleUnsupportedCommandSurface(invocation commandruntime.Invocation, cmdInfo commandcatalog.CommandInfo, agent *Agent, commandSurface commandcatalog.CommandSurface) bool {
	if len(invocation.Args) != 0 {
		return false
	}

	yellow.Fprintf(agent.output(), "⚠️  %s is available in %s mode only.\n", invocation.Command, commandSurfaceHint(cmdInfo))
	if commandSurface == commandcatalog.CommandSurfaceClassic && cmdInfo.SupportsSurface(commandcatalog.CommandSurfaceTUI) {
		yellow.Fprintln(agent.output(), "   Run without --no-tui to use the TUI command.")
	}
	return true
}

func commandSurfaceHint(cmdInfo commandcatalog.CommandInfo) string {
	if cmdInfo.SupportsSurface(commandcatalog.CommandSurfaceTUI) {
		return "TUI"
	}
	if cmdInfo.SupportsSurface(commandcatalog.CommandSurfaceClassic) {
		return "classic"
	}
	return "another"
}

// splitCommand はコマンド文字列を分割する。
func splitCommand(input string) []string {
	return commandruntime.Split(input)
}
