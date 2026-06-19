package agent

// syncCurrentDerivedRuntimeState refreshes runtime state derived from the
// current provider/model selection. This keeps prompt wording and current
// surface visibility aligned immediately after provider/model changes, even
// before the next chat request.
func (a *Agent) syncCurrentDerivedRuntimeState() {
	if a == nil {
		return
	}
	previousSurface, _ := a.refreshMCPToolSurface()
	a.configureCurrentProviderMCPTools()
	a.rebuildSystemPromptForCurrentProvider()
	a.syncCurrentSurfaceToolVisibilityWithPreviousBudget(previousSurface.omittedExportedNames())
}
