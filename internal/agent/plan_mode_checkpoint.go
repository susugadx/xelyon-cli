package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
)

type planModeCheckpoint struct {
	historyLen                      int
	sessionMessageCount             int
	responseID                      string
	systemPrompt                    string
	pendingApprovedPlan             string
	pendingApprovedPlanHasChanges   bool
	pendingApprovedPlanChangedFiles []string
	restored                        bool
}

func capturePlanModeCheckpoint(a *Agent, currentRequest string) planModeCheckpoint {
	checkpoint := planModeCheckpoint{}
	if a == nil {
		return checkpoint
	}

	checkpoint.historyLen = len(a.History)
	checkpoint.systemPrompt = a.SystemPrompt
	checkpoint.pendingApprovedPlan = a.pendingApprovedPlan()
	checkpoint.pendingApprovedPlanHasChanges = a.pendingApprovedPlanHasChanges()
	checkpoint.pendingApprovedPlanChangedFiles = a.pendingApprovedPlanChangedFiles()
	if a.session != nil {
		checkpoint.sessionMessageCount = len(a.session.Messages)
		if shouldExcludeCurrentPlanRequest(a.session, currentRequest) {
			checkpoint.sessionMessageCount--
		}
	}
	if ridProvider, ok := a.CurrentProvider.(ResponseIDCapable); ok {
		checkpoint.responseID = ridProvider.GetResponseID()
	}
	return checkpoint
}

func (c *planModeCheckpoint) restore(a *Agent, restoreApprovedPlan bool) error {
	if c == nil || c.restored || a == nil {
		return nil
	}
	c.restored = true

	if ridProvider, ok := a.CurrentProvider.(ResponseIDCapable); ok {
		ridProvider.SetResponseID(c.responseID)
	}
	a.SystemPrompt = c.systemPrompt
	if restoreApprovedPlan {
		a.PendingApprovedPlan = c.pendingApprovedPlan
		a.PendingApprovedPlanHasChanges = c.pendingApprovedPlanHasChanges && c.pendingApprovedPlan != ""
		a.PendingApprovedPlanChangedFiles = clonePendingApprovedPlanChangedFiles(c.pendingApprovedPlanChangedFiles)
	}

	if c.historyLen >= 0 && c.historyLen <= len(a.History) {
		a.History = append([]api.Message(nil), a.History[:c.historyLen]...)
	}

	if a.session != nil {
		truncated := a.session.TruncateMessages(c.sessionMessageCount)
		a.session.ResponseID = c.responseID
		if restoreApprovedPlan {
			a.session.PendingApprovedPlan = c.pendingApprovedPlan
			a.session.PendingApprovedPlanHasChanges = c.pendingApprovedPlanHasChanges && c.pendingApprovedPlan != ""
			a.session.PendingApprovedPlanChangedFiles = clonePendingApprovedPlanChangedFiles(c.pendingApprovedPlanChangedFiles)
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
