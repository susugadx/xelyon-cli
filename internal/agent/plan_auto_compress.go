package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func (r *planModeRequest) investigationAutoCompression() planInvestigationAutoCompression {
	return planInvestigationAutoCompression{
		currentTurnStartIndex: r.checkpoint.historyLen,
		turnState:             r.autoCompression,
		persistHistory:        r.persistableHistoryForInTurnCompression,
		onSuccess:             r.handleInTurnCompressionSuccess,
	}
}

func (r *planModeRequest) handleInTurnCompressionSuccess(currentTurnStartIndex int) {
	if r == nil || r.agent == nil || currentTurnStartIndex <= 0 {
		return
	}
	r.checkpoint.rebaseAfterInTurnCompression(r.agent, currentTurnStartIndex)
}

func (r *planModeRequest) persistableHistoryForInTurnCompression() []api.Message {
	if r == nil || r.agent == nil {
		return nil
	}
	a := r.agent
	runtimeHistory := stripNormalModePromptFromHistory(a.History)
	if len(runtimeHistory) == 0 {
		return nil
	}

	if a.session != nil {
		sessionHistory := a.session.ToAPIMessages()
		if r.historiesMatchForPlanCompressionPersistence(runtimeHistory, sessionHistory) {
			return sessionHistory
		}
	}

	persistHistory := append([]api.Message(nil), runtimeHistory...)
	r.replaceInvestigationPromptForPersistence(persistHistory)
	return persistHistory
}

func (r *planModeRequest) historiesMatchForPlanCompressionPersistence(runtimeHistory, sessionHistory []api.Message) bool {
	if len(runtimeHistory) != len(sessionHistory) {
		return false
	}
	for i := range runtimeHistory {
		if i == r.checkpoint.historyLen && r.isPersistedCurrentUserRequest(sessionHistory[i]) {
			continue
		}
		if !messagesMatchForCompressionPersistence(runtimeHistory[i], sessionHistory[i]) {
			return false
		}
	}
	return true
}

func (r *planModeRequest) isPersistedCurrentUserRequest(msg api.Message) bool {
	return msg.Role == "user" && strings.TrimSpace(msg.Content) == strings.TrimSpace(r.originalUserRequest)
}

func (r *planModeRequest) replaceInvestigationPromptForPersistence(history []api.Message) {
	idx := r.checkpoint.historyLen
	if idx < 0 || idx >= len(history) || history[idx].Role != "user" {
		return
	}
	history[idx].Content = r.originalUserRequest
}
