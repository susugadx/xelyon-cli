package agent

import (
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/commandruntime"
	"github.com/susugadx/xelyon-cli/internal/version"
)

func specialCommandRegistry(agent *Agent, commandSurface commandcatalog.CommandSurface) commandruntime.Registry {
	return commandruntime.Registry{
		"/save":      func(_ []string) bool { return handleSaveCommand(agent) },
		"/load":      func(args []string) bool { return handleLoadCommand(agent, args) },
		"/sessions":  func(_ []string) bool { return handleSessionsCommand(agent) },
		"/config":    func(args []string) bool { return handleConfigCommand(agent, args) },
		"/stats":     func(_ []string) bool { return handleStatsCommand(agent) },
		"/status":    func(_ []string) bool { return handleStatusCommand(agent) },
		"/copy":      func(args []string) bool { return handleCopyCommand(agent, args) },
		"/compress":  func(args []string) bool { return handleCompressCommand(agent, args) },
		"/use":       func(args []string) bool { return handleUseCommand(agent, args) },
		"/providers": func(_ []string) bool { return handleProvidersCommand(agent) },
		"/exit":      func(_ []string) bool { handleExitCommand(agent); return true },
		"/quit":      func(_ []string) bool { handleExitCommand(agent); return true },
		"/q":         func(_ []string) bool { handleExitCommand(agent); return true },
		"/clear":     func(args []string) bool { return handleClearCommand(agent, args) },
		"/history":   func(_ []string) bool { handleHistoryCommand(agent); return true },
		"/help":      func(_ []string) bool { printHelpToWriterForSurface(agent.output(), agent, commandSurface); return true },
		"/model":     func(args []string) bool { return handleModelCommand(agent, args) },
		"/version":   func(args []string) bool { return handleVersionCommand(agent, args) },
		"/plan":      func(args []string) bool { return handlePlanCommand(agent, args) },
		"/init": func(_ []string) bool {
			return handleInitCommandWithOptions(agent, initCommandOptions{
				allowOverwritePrompt: commandSurface != commandcatalog.CommandSurfaceTUI,
			})
		},
		"/project": func(_ []string) bool { return handleProjectCommand(agent) },
		"/lsp":     func(args []string) bool { return handleLSPCommand(agent, args) },
		"/tokens":  func(_ []string) bool { return handleTokensCommand(agent) },
		"/think":   func(args []string) bool { return handleThinkCommand(agent, args) },
	}
}

func handleClearCommand(agent *Agent, _ []string) bool {
	if err := agent.resetConversationState(); err != nil {
		red.Fprintf(agent.output(), "Failed to clear history: %v\n", err)
		return true
	}
	green.Fprintln(agent.output(), "🗑️  History cleared")
	return true
}

func handleVersionCommand(agent *Agent, _ []string) bool {
	cyan.Fprintf(agent.output(), "🚀 XELYON CLI v%s\n", version.GetVersion())
	return true
}
