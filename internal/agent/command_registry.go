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
		"/new":       func(_ []string) bool { return handleNewCommand(agent) },
		"/resume":    func(args []string) bool { return handleResumeCommand(agent, args) },
		"/config":    func(args []string) bool { return handleConfigCommandForSurface(agent, args, commandSurface) },
		"/stats":     func(_ []string) bool { return handleStatsCommandForSurface(agent, commandSurface) },
		"/status":    func(_ []string) bool { return handleStatusCommandForSurface(agent, commandSurface) },
		"/copy":      func(args []string) bool { return handleCopyCommand(agent, args) },
		"/compress":  func(args []string) bool { return handleCompressCommand(agent, args) },
		"/provider":  func(args []string) bool { return handleProviderCommand(agent, args) },
		"/use":       func(args []string) bool { return handleUseCommand(agent, args) },
		"/providers": func(_ []string) bool { return handleProvidersCommand(agent) },
		"/skills":    func(args []string) bool { return handleSkillsCommand(agent, args) },
		"/exit":      func(_ []string) bool { handleExitCommand(agent); return true },
		"/quit":      func(_ []string) bool { handleExitCommand(agent); return true },
		"/q":         func(_ []string) bool { handleExitCommand(agent); return true },
		"/clear":     func(args []string) bool { return handleClearCommand(agent, args) },
		"/history":   func(_ []string) bool { handleHistoryCommand(agent); return true },
		"/help":      func(_ []string) bool { printHelpToWriterForSurface(agent.output(), agent, commandSurface); return true },
		"/h":         func(_ []string) bool { printHelpToWriterForSurface(agent.output(), agent, commandSurface); return true },
		"/model":     func(args []string) bool { return handleModelCommand(agent, args) },
		"/version":   func(args []string) bool { return handleVersionCommand(agent, args) },
		"/plan":      func(args []string) bool { return handlePlanCommand(agent, args) },
		"/init": func(_ []string) bool {
			return handleInitCommandWithOptions(agent, initCommandOptions{
				allowOverwritePrompt: commandSurface != commandcatalog.CommandSurfaceTUI,
			})
		},
		"/tokens":     func(_ []string) bool { return handleTokensCommand(agent) },
		"/ledger":     func(args []string) bool { return handleLedgerCommand(agent, args) },
		"/rawoutputs": func(args []string) bool { return handleRawOutputsCommand(agent, args) },
		"/thinking":   func(args []string) bool { return handleThinkingCommand(agent, args) },
		"/think":      func(args []string) bool { return handleThinkingCommand(agent, args) },
	}
}

func handleClearCommand(agent *Agent, _ []string) bool {
	session, err := agent.StartNewSession()
	if err != nil {
		red.Fprintf(agent.output(), "Failed to start new session: %v\n", err)
		return true
	}
	green.Fprintf(agent.output(), "🗑️  History cleared; started new session %s\n", session.ID)
	return true
}

func handleNewCommand(agent *Agent) bool {
	if session, err := agent.StartNewSession(); err != nil {
		red.Fprintf(agent.output(), "Failed to start new session: %v\n", err)
	} else if session != nil {
		green.Fprintf(agent.output(), "🆕 Started new session %s\n", session.ID)
	}
	return true
}

func handleVersionCommand(agent *Agent, _ []string) bool {
	cyan.Fprintf(agent.output(), "🚀 XELYON CLI v%s\n", version.GetVersion())
	return true
}
