package agent

// syncCurrentDerivedRuntimeState refreshes runtime state derived from the
// current provider/model selection. This keeps prompt wording and current
// surface visibility aligned immediately after provider/model changes, even
// before the next chat request.
func (a *Agent) syncCurrentDerivedRuntimeState() {
	if a == nil {
		return
	}
	a.rebuildSystemPromptForCurrentProvider()
	a.syncCurrentSurfaceToolVisibility()
}
