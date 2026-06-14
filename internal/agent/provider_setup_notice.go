package agent

import "github.com/susugadx/xelyon-cli/internal/api"

func printProviderSetupRequiredNotice(agent *Agent) {
	if agent == nil {
		return
	}
	if msg, ok := api.ProviderSetupRequiredMessage(agent.CurrentProvider); ok {
		yellow.Fprintf(agent.output(), "⚠️  %s\n\n", msg)
	}
}
