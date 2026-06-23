package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
)

type responseContextSnapshot struct {
	responseID        string
	model             string
	providerName      string
	providerConfigKey string
	promptFingerprint string
}

func responseContextSnapshotFromSession(session *history.Session) responseContextSnapshot {
	if session == nil {
		return responseContextSnapshot{}
	}

	snapshot := responseContextSnapshot{
		responseID:        strings.TrimSpace(session.ResponseID),
		model:             strings.TrimSpace(session.ResponseModel),
		providerName:      strings.TrimSpace(session.ResponseProviderName),
		providerConfigKey: strings.TrimSpace(session.ResponseProviderConfigKey),
		promptFingerprint: strings.TrimSpace(session.ResponsePromptFingerprint),
	}

	if snapshot.model == "" {
		snapshot.model = strings.TrimSpace(session.Model)
	}
	if snapshot.providerName == "" {
		snapshot.providerName = strings.TrimSpace(session.ProviderName)
	}
	if snapshot.providerConfigKey == "" {
		snapshot.providerConfigKey = strings.TrimSpace(session.ProviderConfigKey)
	}

	return snapshot
}

func responseContextSnapshotFromRuntime(model, providerName, providerConfigKey, responseID string) responseContextSnapshot {
	return responseContextSnapshot{
		responseID:        strings.TrimSpace(responseID),
		model:             strings.TrimSpace(model),
		providerName:      config.CanonicalProviderName(providerName),
		providerConfigKey: config.ActiveProviderConfigKey(providerConfigKey),
	}
}

func (snapshot responseContextSnapshot) hasResponseID() bool {
	return strings.TrimSpace(snapshot.responseID) != ""
}

func (snapshot responseContextSnapshot) shouldRestoreForRuntime(currentModel, currentProviderName, currentProviderConfigKey string) bool {
	return shouldRestoreSessionResponseID(
		snapshot.model,
		currentModel,
		snapshot.providerName,
		currentProviderName,
		snapshot.providerConfigKey,
		currentProviderConfigKey,
		snapshot.responseID,
	)
}

func (snapshot responseContextSnapshot) shouldReuseForRuntimePrompt(currentModel, currentProviderName, currentProviderConfigKey, promptFingerprint string) bool {
	if !snapshot.shouldRestoreForRuntime(currentModel, currentProviderName, currentProviderConfigKey) {
		return false
	}
	promptFingerprint = strings.TrimSpace(promptFingerprint)
	return promptFingerprint != "" && snapshot.promptFingerprint == promptFingerprint
}

func (snapshot responseContextSnapshot) applyToSession(session *history.Session) {
	if session == nil {
		return
	}
	session.ApplyResponseContextWithFingerprint(
		snapshot.responseID,
		snapshot.model,
		config.CanonicalProviderName(snapshot.providerName),
		config.ActiveProviderConfigKey(snapshot.providerConfigKey),
		snapshot.promptFingerprint,
	)
}

func (snapshot responseContextSnapshot) applyToProvider(provider ResponseIDCapable) {
	if provider == nil {
		return
	}
	provider.SetResponseID(strings.TrimSpace(snapshot.responseID))
}

func clearSavedResponseContext(session *history.Session) {
	if session == nil {
		return
	}
	session.ClearResponseContext()
}

func (a *Agent) clearProviderAndSavedResponseContext() {
	if a == nil {
		return
	}
	if ridProvider, ok := a.CurrentProvider.(ResponseIDCapable); ok {
		ridProvider.SetResponseID("")
	}
	a.responseContext = responseContextSnapshot{}
	clearSavedResponseContext(a.session)
}

func (a *Agent) hasResponseIDChainProvider() bool {
	if a == nil {
		return false
	}
	_, ok := a.CurrentProvider.(ResponseIDCapable)
	return ok
}
