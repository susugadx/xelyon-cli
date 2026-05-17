package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
)

type planModeCheckpoint struct {
	historyLen          int
	sessionMessageCount int
	responseID          string
	responseModel       string
	responseProvider    string
	responseProviderKey string
	systemPrompt        string
	restored            bool
}

func capturePlanModeCheckpoint(a *Agent, currentRequest string) planModeCheckpoint {
	checkpoint := planModeCheckpoint{}
	if a == nil {
		return checkpoint
	}

	checkpoint.historyLen = len(a.History)
	checkpoint.systemPrompt = a.SystemPrompt
	checkpoint.responseModel = a.CurrentModel
	checkpoint.responseProvider = config.CanonicalProviderName(a.ProviderName)
	checkpoint.responseProviderKey = a.currentProviderConfigKey()
	if a.session != nil {
		checkpoint.sessionMessageCount = len(a.session.Messages)
		if shouldExcludeCurrentPlanRequest(a.session, currentRequest) {
			checkpoint.sessionMessageCount--
		}
	}
	if a.responsesPersistResponseIDEnabled() && a.session != nil {
		snapshot := responseContextSnapshotFromSession(a.session)
		if snapshot.hasResponseID() {
			checkpoint.applyResponseContextSnapshot(snapshot)
		}
	}
	if ridProvider, ok := a.CurrentProvider.(ResponseIDCapable); ok && a.responsesStoreEnabled() {
		checkpoint.responseID = ridProvider.GetResponseID()
		if checkpoint.responseID != "" {
			checkpoint.applyResponseContextSnapshot(responseContextSnapshotFromRuntime(
				a.CurrentModel,
				a.ProviderName,
				a.currentProviderConfigKey(),
				checkpoint.responseID,
			))
		}
	}
	return checkpoint
}

func (c *planModeCheckpoint) restore(a *Agent) error {
	if c == nil || c.restored || a == nil {
		return nil
	}
	c.restored = true

	c.restoreProviderResponseContext(a)
	a.SystemPrompt = c.systemPrompt

	resetTaskLedger := false
	defer func() {
		if resetTaskLedger {
			a.resetProviderFacingTaskLedger()
		}
	}()

	if c.historyLen >= 0 && c.historyLen <= len(a.History) {
		resetTaskLedger = c.historyLen < len(a.History)
		a.History = append([]api.Message(nil), a.History[:c.historyLen]...)
	}

	if a.session != nil {
		truncated := a.session.TruncateMessages(c.sessionMessageCount)
		resetTaskLedger = resetTaskLedger || truncated
		if a.responsesPersistResponseIDEnabled() {
			c.restoreSessionResponseContext(a.session)
		} else {
			clearSavedResponseContext(a.session)
		}
		if truncated {
			if a.storage != nil {
				if err := a.storage.Rewrite(a.session); err != nil {
					return err
				}
			}
		} else {
			a.persistSession()
		}
	}

	return nil
}

func (c *planModeCheckpoint) restoreSystemPrompt(a *Agent) {
	if c == nil || a == nil {
		return
	}
	a.SystemPrompt = c.systemPrompt
}

func (c *planModeCheckpoint) restoreProviderResponseContext(a *Agent) {
	if a == nil {
		return
	}
	if ridProvider, ok := a.CurrentProvider.(ResponseIDCapable); ok {
		if !a.responsesStoreEnabled() {
			ridProvider.SetResponseID("")
			return
		}
		ridProvider.SetResponseID(strings.TrimSpace(c.responseID))
	}
}

func (c *planModeCheckpoint) restoreSessionResponseContext(session *history.Session) {
	c.responseContextSnapshot().applyToSession(session)
}

func (c *planModeCheckpoint) responseContextSnapshot() responseContextSnapshot {
	if c == nil {
		return responseContextSnapshot{}
	}
	return responseContextSnapshot{
		responseID:        strings.TrimSpace(c.responseID),
		model:             strings.TrimSpace(c.responseModel),
		providerName:      strings.TrimSpace(c.responseProvider),
		providerConfigKey: strings.TrimSpace(c.responseProviderKey),
	}
}

func (c *planModeCheckpoint) applyResponseContextSnapshot(snapshot responseContextSnapshot) {
	if c == nil {
		return
	}
	c.responseID = strings.TrimSpace(snapshot.responseID)
	c.responseModel = strings.TrimSpace(snapshot.model)
	c.responseProvider = config.CanonicalProviderName(snapshot.providerName)
	c.responseProviderKey = config.ActiveProviderConfigKey(snapshot.providerConfigKey)
}

func shouldExcludeCurrentPlanRequest(session *history.Session, currentRequest string) bool {
	currentRequest = strings.TrimSpace(currentRequest)
	if currentRequest == "" {
		return false
	}

	if session == nil || len(session.Messages) == 0 {
		return false
	}
	last := session.Messages[len(session.Messages)-1]
	return last.Role == "user" && strings.TrimSpace(last.Content) == currentRequest
}
