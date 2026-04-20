package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
)

// handleTextPlanRecovery は normal mode で no-tool 応答が
// 実行ではなく計画文へ逸れた時の補正だけを担当する。
// wording 判定は使わず、no-mutation + strong plan signal の場合だけ補正する。
func (h *normalModeNoToolHandler) handleTextPlanRecovery(response string) (bool, normalModeAction) {
	a := h.runner.agent
	if !hasStrongTextPlanSignal(response) {
		return false, normalModeContinue
	}

	steps := extractTextPlan(response)
	if len(steps) < 3 || !isActionPlan(steps) {
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

func hasStrongTextPlanSignal(response string) bool {
	trimmed := strings.TrimSpace(response)
	if trimmed == "" {
		return false
	}
	if plan.ContainsPlanJSON(trimmed) {
		return true
	}

	lowered := strings.ToLower(trimmed)
	strongPrefixes := []string{
		"here is the plan",
		"plan:",
		"plan：",
		"execution plan:",
		"execution plan：",
		"計画:",
		"計画：",
		"実行計画:",
		"実行計画：",
		"作業計画:",
		"作業計画：",
		"# plan",
		"## plan",
		"### plan",
	}
	for _, prefix := range strongPrefixes {
		if strings.HasPrefix(lowered, prefix) {
			return true
		}
	}

	markers := []string{
		"\nhere is the plan",
		"\nplan:",
		"\nplan：",
		"\nexecution plan:",
		"\nexecution plan：",
		"\n計画:",
		"\n計画：",
		"\n実行計画:",
		"\n実行計画：",
		"\n作業計画:",
		"\n作業計画：",
		"\n# plan",
		"\n## plan",
		"\n### plan",
	}
	for _, marker := range markers {
		if strings.Contains(lowered, marker) {
			return true
		}
	}

	return strings.Contains(lowered, "```json") && strings.Contains(lowered, "\"plan\"")
}
