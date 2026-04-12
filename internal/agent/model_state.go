package agent

// setCurrentModel updates the agent's current model and the directly coupled
// session/stats mirrors owned by the interactive runtime state.
func (a *Agent) setCurrentModel(model string) {
	if a == nil {
		return
	}
	a.CurrentModel = model
	a.syncSessionModel()
	if a.Stats != nil {
		a.statsMu.Lock()
		a.Stats.Model = model
		a.statsMu.Unlock()
	}
}

// setCurrentModelAndSync updates the current model and immediately refreshes
// derived runtime state that depends on provider/model selection.
func (a *Agent) setCurrentModelAndSync(model string) {
	if a == nil {
		return
	}
	a.setCurrentModel(model)
	a.syncCurrentDerivedRuntimeState()
}
