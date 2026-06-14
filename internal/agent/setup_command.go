package agent

import (
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/setup"
)

func handleSetupCommandForSurface(agent *Agent, commandSurface commandcatalog.CommandSurface) bool {
	if agent == nil {
		return true
	}
	opts := setup.Options{
		Config:   agent.cfg(),
		CWD:      agent.invocationCWD(),
		Provider: agent.ProviderName,
		Model:    agent.CurrentModel,
	}
	if commandSurface == commandcatalog.CommandSurfaceTUI {
		opts.ProjectConfigInstructionMode = setup.ProjectConfigInstructionTUI
	}
	setup.Render(agent.output(), setup.BuildReport(opts))
	return true
}
