package agent

func (a *Agent) invocationCWD() string {
	if a == nil || a.Runtime == nil {
		return resolveRuntimeInvocationCWD()
	}
	return a.Runtime.effectiveInvocationCWD()
}

func (a *Agent) projectConfigStore() *ProjectConfigStore {
	if a == nil || a.Runtime == nil {
		return defaultProjectConfigStore
	}
	return a.Runtime.effectiveProjectConfigStore()
}
