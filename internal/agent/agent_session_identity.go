package agent

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
)

func (a *Agent) syncSessionRuntimeIdentity() {
	if a == nil || a.session == nil {
		return
	}

	a.session.Model = a.CurrentModel
	a.session.ProviderName = config.CanonicalProviderName(a.ProviderName)
	a.session.ProviderConfigKey = a.currentProviderConfigKey()
}

func (a *Agent) syncSessionPersistenceState() {
	if a == nil || a.session == nil {
		return
	}

	a.syncApprovedPlanStateToSession()
	a.syncSessionRuntimeIdentity()
	a.syncSavedResponseContextFromProvider()
}

func (a *Agent) reconcileSessionForCurrentRuntime() {
	if a == nil || a.session == nil {
		return
	}

	a.syncSessionRuntimeIdentity()
	a.restoreSessionResponseIDForCurrentContext()
}

func savedResponseContext(session *history.Session) (model, providerName, providerConfigKey, responseID string) {
	if session == nil {
		return "", "", "", ""
	}
	model = session.ResponseModel
	providerName = session.ResponseProviderName
	providerConfigKey = session.ResponseProviderConfigKey
	responseID = session.ResponseID
	if model == "" {
		model = session.Model
	}
	if providerName == "" {
		providerName = session.ProviderName
	}
	if providerConfigKey == "" {
		providerConfigKey = session.ProviderConfigKey
	}
	return model, providerName, providerConfigKey, responseID
}

func shouldRestoreSessionResponseID(sessionModel, currentModel, sessionProviderName, currentProviderName, sessionProviderConfigKey, currentProviderConfigKey, responseID string) bool {
	if responseID == "" {
		return false
	}
	if sessionModel == "" || currentModel == "" || sessionModel != currentModel {
		return false
	}

	normalizedCurrentProvider := config.CanonicalProviderName(currentProviderName)
	normalizedCurrentProviderConfigKey := config.NormalizeProviderName(currentProviderConfigKey)
	normalizedSessionProvider := config.CanonicalProviderName(sessionProviderName)
	normalizedSessionProviderConfigKey := config.NormalizeProviderName(sessionProviderConfigKey)

	if normalizedSessionProvider == "" && normalizedSessionProviderConfigKey == "" {
		return false
	}

	if normalizedSessionProvider != "" && !config.SameProviderRuntimeIdentity(normalizedSessionProvider, normalizedCurrentProvider) {
		return false
	}
	if normalizedSessionProviderConfigKey != "" && normalizedCurrentProviderConfigKey != "" && normalizedSessionProviderConfigKey != normalizedCurrentProviderConfigKey {
		return false
	}
	return true
}

func (a *Agent) syncSavedResponseContextFromProvider() {
	if a == nil || a.session == nil {
		return
	}

	ridProvider, ok := a.CurrentProvider.(ResponseIDCapable)
	if !ok || !ridProvider.HasCachedResponseID() {
		return
	}

	a.session.ResponseID = ridProvider.GetResponseID()
	a.session.ResponseModel = a.CurrentModel
	a.session.ResponseProviderName = config.CanonicalProviderName(a.ProviderName)
	a.session.ResponseProviderConfigKey = a.currentProviderConfigKey()
}

func clearSavedResponseContext(session *history.Session) {
	if session == nil {
		return
	}

	session.ResponseID = ""
	session.ResponseModel = ""
	session.ResponseProviderName = ""
	session.ResponseProviderConfigKey = ""
}

func (a *Agent) invalidateSavedResponseContextForCurrentRuntime() {
	if a == nil || a.session == nil {
		return
	}

	sessionModel, sessionProviderName, sessionProviderConfigKey, responseID := savedResponseContext(a.session)
	if responseID == "" {
		return
	}
	if !shouldRestoreSessionResponseID(
		sessionModel,
		a.CurrentModel,
		sessionProviderName,
		a.ProviderName,
		sessionProviderConfigKey,
		a.currentProviderConfigKey(),
		responseID,
	) {
		clearSavedResponseContext(a.session)
	}
}

func (a *Agent) restoreSessionResponseIDForCurrentContext() {
	if a == nil || a.session == nil {
		return
	}

	ridProvider, ok := a.CurrentProvider.(ResponseIDCapable)
	if !ok {
		return
	}

	sessionModel, sessionProviderName, sessionProviderConfigKey, responseID := savedResponseContext(a.session)
	if !shouldRestoreSessionResponseID(
		sessionModel,
		a.CurrentModel,
		sessionProviderName,
		a.ProviderName,
		sessionProviderConfigKey,
		a.currentProviderConfigKey(),
		responseID,
	) {
		ridProvider.SetResponseID("")
		return
	}

	ridProvider.SetResponseID(responseID)
}

func (a *Agent) applyLoadedSession(session *history.Session) {
	if a == nil {
		return
	}

	a.restoreSessionConversation(session)
	a.reconcileSessionForCurrentRuntime()
}
