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
		sessionModel, sessionProvider, sessionProviderKey, sessionResponseID := savedResponseContext(a.session)
		if sessionResponseID != "" {
			checkpoint.responseID = sessionResponseID
			checkpoint.responseModel = sessionModel
			checkpoint.responseProvider = sessionProvider
			checkpoint.responseProviderKey = sessionProviderKey
		}
	}
	if ridProvider, ok := a.CurrentProvider.(ResponseIDCapable); ok {
		checkpoint.responseID = ridProvider.GetResponseID()
		if checkpoint.responseID != "" {
			checkpoint.responseModel = a.CurrentModel
			checkpoint.responseProvider = config.CanonicalProviderName(a.ProviderName)
			checkpoint.responseProviderKey = a.currentProviderConfigKey()
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

	if c.historyLen >= 0 && c.historyLen <= len(a.History) {
		a.History = append([]api.Message(nil), a.History[:c.historyLen]...)
	}

	if a.session != nil {
		truncated := a.session.TruncateMessages(c.sessionMessageCount)
		c.restoreSessionResponseContext(a.session)
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
		ridProvider.SetResponseID(strings.TrimSpace(c.responseID))
	}
}

func (c *planModeCheckpoint) restoreSessionResponseContext(session *history.Session) {
	if session == nil {
		return
	}

	responseID := strings.TrimSpace(c.responseID)
	if responseID == "" {
		clearSavedResponseContext(session)
		return
	}

	session.ResponseID = responseID
	session.ResponseModel = strings.TrimSpace(c.responseModel)
	if session.ResponseModel == "" {
		session.ResponseModel = session.Model
	}

	session.ResponseProviderName = config.CanonicalProviderName(strings.TrimSpace(c.responseProvider))
	if session.ResponseProviderName == "" {
		session.ResponseProviderName = config.CanonicalProviderName(session.ProviderName)
	}

	session.ResponseProviderConfigKey = config.NormalizeProviderName(strings.TrimSpace(c.responseProviderKey))
	if session.ResponseProviderConfigKey == "" {
		session.ResponseProviderConfigKey = session.ProviderConfigKey
	}
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
