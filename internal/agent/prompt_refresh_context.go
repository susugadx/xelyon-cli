package agent

import "github.com/susugadx/xelyon-cli/internal/config"

type promptRefreshContext struct {
	invocationCWD string
	projectConfig *config.ProjectConfig
}

func newPromptRefreshContext(agent *Agent) promptRefreshContext {
	if agent == nil {
		return promptRefreshContext{}
	}
	cwd := agent.invocationCWD()
	return promptRefreshContext{
		invocationCWD: cwd,
		projectConfig: agent.projectConfigStore().LoadForCWD(cwd),
	}
}
