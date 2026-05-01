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

func shouldRestoreSessionResponseID(sessionModel, currentModel, sessionProviderName, currentProviderName, sessionProviderConfigKey, currentProviderConfigKey, responseID string) bool {
	if responseID == "" {
		return false
	}
	if sessionModel == "" || currentModel == "" || sessionModel != currentModel {
		return false
	}

	normalizedCurrentProvider := config.CanonicalProviderName(currentProviderName)
	normalizedCurrentProviderConfigKey := config.ActiveProviderConfigKey(currentProviderConfigKey)
	normalizedSessionProvider := config.CanonicalProviderName(sessionProviderName)
	normalizedSessionProviderConfigKey := config.ActiveProviderConfigKey(sessionProviderConfigKey)

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

func (a *Agent) responsesStoreEnabled() bool {
	return a.cfg().ResponsesStoreEnabled()
}

func (a *Agent) responsesPersistResponseIDEnabled() bool {
	return a.cfg().ResponsesPersistResponseIDEnabled()
}

func (a *Agent) syncSavedResponseContextFromProvider() {
	if a == nil || a.session == nil {
		return
	}
	if !a.responsesPersistResponseIDEnabled() {
		clearSavedResponseContext(a.session)
		return
	}

	ridProvider, ok := a.CurrentProvider.(ResponseIDCapable)
	if !ok || !ridProvider.HasCachedResponseID() {
		return
	}

	responseContextSnapshotFromRuntime(
		a.CurrentModel,
		a.ProviderName,
		a.currentProviderConfigKey(),
		ridProvider.GetResponseID(),
	).applyToSession(a.session)
}

func (a *Agent) invalidateSavedResponseContextForCurrentRuntime() {
	if a == nil || a.session == nil {
		return
	}
	if !a.responsesPersistResponseIDEnabled() {
		clearSavedResponseContext(a.session)
		return
	}

	snapshot := responseContextSnapshotFromSession(a.session)
	if !snapshot.hasResponseID() {
		return
	}
	if !snapshot.shouldRestoreForRuntime(
		a.CurrentModel,
		a.ProviderName,
		a.currentProviderConfigKey(),
	) {
		clearSavedResponseContext(a.session)
	}
}

func (a *Agent) restoreSessionResponseIDForCurrentContext() {
	if a == nil || a.session == nil {
		return
	}

	if !a.responsesPersistResponseIDEnabled() {
		if ridProvider, ok := a.CurrentProvider.(ResponseIDCapable); ok {
			ridProvider.SetResponseID("")
		}
		clearSavedResponseContext(a.session)
		return
	}

	ridProvider, ok := a.CurrentProvider.(ResponseIDCapable)
	if !ok {
		return
	}

	snapshot := responseContextSnapshotFromSession(a.session)
	if !snapshot.shouldRestoreForRuntime(
		a.CurrentModel,
		a.ProviderName,
		a.currentProviderConfigKey(),
	) {
		ridProvider.SetResponseID("")
		return
	}

	snapshot.applyToProvider(ridProvider)
}

func (a *Agent) applyLoadedSession(session *history.Session) {
	if a == nil {
		return
	}

	a.restoreSessionConversation(session)
	a.reconcileSessionForCurrentRuntime()
}
