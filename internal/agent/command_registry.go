package agent

import "github.com/susugadx/xelyon-cli/internal/version"

type specialCommandHandler func(*Agent, []string) bool

func specialCommandRegistry() map[string]specialCommandHandler {
	return map[string]specialCommandHandler{
		"/save":      func(agent *Agent, _ []string) bool { return handleSaveCommand(agent) },
		"/load":      handleLoadCommand,
		"/sessions":  func(agent *Agent, _ []string) bool { return handleSessionsCommand(agent) },
		"/config":    handleConfigCommand,
		"/stats":     func(agent *Agent, _ []string) bool { return handleStatsCommand(agent) },
		"/status":    func(agent *Agent, _ []string) bool { return handleStatusCommand(agent) },
		"/copy":      handleCopyCommand,
		"/compress":  handleCompressCommand,
		"/use":       handleUseCommand,
		"/providers": func(agent *Agent, _ []string) bool { return handleProvidersCommand(agent) },
		"/exit":      func(agent *Agent, _ []string) bool { handleExitCommand(agent); return true },
		"/quit":      func(agent *Agent, _ []string) bool { handleExitCommand(agent); return true },
		"/q":         func(agent *Agent, _ []string) bool { handleExitCommand(agent); return true },
		"/clear":     handleClearCommand,
		"/history":   func(agent *Agent, _ []string) bool { handleHistoryCommand(agent); return true },
		"/help":      func(agent *Agent, _ []string) bool { printHelpToWriter(agent.output(), agent); return true },
		"/model":     handleModelCommand,
		"/version":   handleVersionCommand,
		"/plan":      handlePlanCommand,
		"/init":      func(agent *Agent, _ []string) bool { return handleInitCommand(agent) },
		"/project":   func(agent *Agent, _ []string) bool { return handleProjectCommand(agent) },
		"/paste":     handlePasteCommand,
		"/lsp":       handleLSPCommand,
		"/tokens":    func(agent *Agent, _ []string) bool { return handleTokensCommand(agent) },
		"/think":     handleThinkCommand,
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
