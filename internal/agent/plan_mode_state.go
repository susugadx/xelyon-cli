package agent

// setPlanModeEnabled updates the current plan-mode state and immediately
// re-syncs the current surface visibility derived from that state. This keeps
// command-time registry consumers aligned with the active mode before the next
// chat request.
func (a *Agent) setPlanModeEnabled(enabled bool) {
	if a == nil {
		return
	}
	a.PlanModeEnabled = enabled
	a.syncCurrentSurfaceToolVisibility()
}
