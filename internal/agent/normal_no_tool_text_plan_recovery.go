package agent

import "github.com/susugadx/xelyon-cli/internal/api"

// handleTextPlanRecovery は normal mode で no-tool 応答が
// 実行ではなく計画文へ逸れた時の補正だけを担当する。
func (h *normalModeNoToolHandler) handleTextPlanRecovery(response string) (bool, normalModeAction) {
	a := h.runner.agent
	steps := extractTextPlan(response)
	if isCompletionTriggerResponse(response) || len(steps) < 5 || !isActionPlan(steps) {
		return false, normalModeContinue
	}

	h.state.textPlanRedirectCount++
	if h.state.textPlanRedirectCount > maxTextPlanHardLimit {
		yellow.Fprintf(a.output(), "⚠️  Text plan detected %d times without tool use. Returning response to user.\n", h.state.textPlanRedirectCount)
		h.runner.appendAssistantHistoryOnly(response)
		h.state.fallbackResponse = response
		return true, normalModeBreak
	}

	if h.state.textPlanRedirectCount > maxTextPlanRedirects {
		yellow.Fprintf(a.output(), "⚠️  Text plan detected %d times. Forcing direct execution.\n", h.state.textPlanRedirectCount)
		h.runner.appendAssistantHistoryOnly(response)
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: a.toolVisibilityPolicy(toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true}).normalModeRecoveryPrompt(normalModeRecoveryPromptStopPlanning),
		})
		return true, normalModeContinue
	}

	yellow.Fprintf(a.output(), "⚠️  Text plan detected (%d steps). Execute tools directly instead. (%d/%d)\n",
		len(steps), h.state.textPlanRedirectCount, maxTextPlanRedirects)
	h.runner.appendAssistantHistoryOnly(response)
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: a.toolVisibilityPolicy(toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true}).normalModeRecoveryPrompt(normalModeRecoveryPromptNoTextPlan),
	})
	return true, normalModeContinue
}
