package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
)

func (a *Agent) restoreSessionConversation(session *history.Session) {
	if a == nil {
		return
	}

	a.session = session
	a.lastOutputs = nil

	if session == nil {
		a.History = nil
		a.RestoreCompactedState(nil)
		a.restoreProviderResponseID("")
		return
	}

	a.History = session.ToAPIMessages()
	a.RestoreCompactedState(session)
	a.restoreProviderResponseID(session.ResponseID)
}

func (a *Agent) resetConversationState() error {
	if a == nil {
		return nil
	}

	a.History = nil
	a.lastOutputs = nil
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
	if !a.responsesPersistResponseIDEnabled() {
		responseID = ""
	}
	if ridProvider, ok := a.CurrentProvider.(ResponseIDCapable); ok {
		ridProvider.SetResponseID(strings.TrimSpace(responseID))
	}
}

func (a *Agent) clearResponseContinuationContext() {
	if a == nil {
		return
	}

	a.restoreProviderResponseID("")
	if a.session == nil {
		return
	}

	clearSavedResponseContext(a.session)
	a.persistSession()
}

func (a *Agent) persistLocalCompressionSuccess(messages []api.Message) {
	if a == nil {
		return
	}

	a.restoreProviderResponseID("")
	if a.session == nil {
		return
	}

	a.session.ReplaceMessagesFromAPI(messages, a.CurrentModel)
	a.session.SetCompactedState(convertToHistoryCompactedItems(a.compactedItems), a.isCompactedMode)
	clearSavedResponseContext(a.session)
	a.rewriteSessionWithWarning("⚠️  Warning: Failed to save compressed session: %v\n")
}

func (a *Agent) suspendResponseContinuationForLocalCompression(persistOnSuccess bool) func(success bool, messages []api.Message) {
	if a == nil {
		return func(bool, []api.Message) {}
	}

	var ridProvider ResponseIDCapable
	previousResponseID := ""
	if provider, ok := a.CurrentProvider.(ResponseIDCapable); ok {
		ridProvider = provider
		previousResponseID = provider.GetResponseID()
		provider.SetResponseID("")
	}

	return func(success bool, messages []api.Message) {
		if success {
			if persistOnSuccess {
				a.persistLocalCompressionSuccess(messages)
				return
			}
			a.restoreProviderResponseID("")
			return
		}
		if ridProvider != nil {
			ridProvider.SetResponseID(previousResponseID)
		}
	}
}

func (a *Agent) hasConversationState() bool {
	if a == nil {
		return false
	}

	if len(a.History) > 0 || len(a.lastOutputs) > 0 || a.isCompactedMode || len(a.compactedItems) > 0 {
		return true
	}
	if ridProvider, ok := a.CurrentProvider.(ResponseIDCapable); ok && a.responsesStoreEnabled() && ridProvider.HasCachedResponseID() {
		return true
	}
	if a.session == nil {
		return false
	}
	return len(a.session.Messages) > 0 ||
		len(a.session.CompactedItems) > 0 ||
		a.session.IsCompactedMode ||
		(a.responsesPersistResponseIDEnabled() && strings.TrimSpace(a.session.ResponseID) != "")
}
