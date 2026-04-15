package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/history"
)

func (a *Agent) restoreSessionConversation(session *history.Session) {
	if a == nil {
		return
	}

	a.session = session
	a.activeApprovedPlan = ""
	a.lastOutputs = nil

	if session == nil {
		a.History = nil
		a.restoreApprovedPlanStateFromSession()
		a.RestoreCompactedState(nil)
		a.restoreProviderResponseID("")
		return
	}

	a.History = session.ToAPIMessages()
	a.restoreApprovedPlanStateFromSession()
	a.RestoreCompactedState(session)
	a.restoreProviderResponseID(session.ResponseID)
}

func (a *Agent) resetConversationState() error {
	if a == nil {
		return nil
	}

	a.History = nil
	a.lastOutputs = nil
	a.activeApprovedPlan = ""
	a.PendingApprovedPlan = ""
	a.PendingApprovedPlanHasChanges = false
	a.PendingApprovedPlanChangedFiles = nil
	a.RestoreCompactedState(nil)
	a.restoreProviderResponseID("")

	if a.session == nil {
		return nil
	}

	a.session.ResetConversation()
	if a.storage != nil {
		return a.storage.Rewrite(a.session)
	}
	return nil
}

func (a *Agent) restoreProviderResponseID(responseID string) {
	if a == nil {
		return
	}
	if ridProvider, ok := a.CurrentProvider.(ResponseIDCapable); ok {
		ridProvider.SetResponseID(strings.TrimSpace(responseID))
	}
}

func (a *Agent) hasConversationState() bool {
	if a == nil {
		return false
	}

	if len(a.History) > 0 || len(a.lastOutputs) > 0 || strings.TrimSpace(a.PendingApprovedPlan) != "" || a.isCompactedMode || len(a.compactedItems) > 0 {
		return true
	}
	if ridProvider, ok := a.CurrentProvider.(ResponseIDCapable); ok && ridProvider.HasCachedResponseID() {
		return true
	}
	if a.session == nil {
		return false
	}
	return len(a.session.Messages) > 0 ||
		len(a.session.CompactedItems) > 0 ||
		a.session.IsCompactedMode ||
		strings.TrimSpace(a.session.PendingApprovedPlan) != "" ||
		strings.TrimSpace(a.session.ResponseID) != ""
}
