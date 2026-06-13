package agent

import (
	"github.com/susugadx/xelyon-cli/internal/setup"
)

func handleSetupCommand(agent *Agent) bool {
	if agent == nil {
		return true
	}
	setup.Render(agent.output(), setup.BuildReport(setup.Options{
		Config:   agent.cfg(),
		CWD:      agent.invocationCWD(),
		Provider: agent.ProviderName,
		Model:    agent.CurrentModel,
	}))
	return true
}
