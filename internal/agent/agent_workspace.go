package agent

import "github.com/susugadx/xelyon-cli/internal/config"

func (a *Agent) invocationCWD() string {
	if a == nil || a.Runtime == nil {
		return resolveRuntimeInvocationCWD()
	}
	return a.Runtime.effectiveInvocationCWD()
}

func (a *Agent) refreshInvocationCWD() {
	if a == nil || a.Runtime == nil {
		return
	}
	a.Runtime.refreshInvocationCWD()
}

func (a *Agent) projectConfigStore() *ProjectConfigStore {
	if a == nil || a.Runtime == nil {
		return defaultProjectConfigStore
	}
	return a.Runtime.effectiveProjectConfigStore()
}

func (a *Agent) loadProjectConfig() *config.ProjectConfig {
	if a == nil {
		return defaultProjectConfigStore.Load()
	}
	return a.projectConfigStore().LoadForCWD(a.invocationCWD())
}
