package agent

import (
	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type normalModePlanningHandler struct {
	runner *TurnRunner
}

func newNormalModePlanningHandler(r *TurnRunner) *normalModePlanningHandler {
	return &normalModePlanningHandler{runner: r}
}

func (h *normalModePlanningHandler) HandlePlanJSONFallback(response string, toolCalls []*tools.ToolCall) (normalModeAction, bool, error) {
	if len(toolCalls) != 0 || !plan.ContainsPlanJSON(response) {
		return normalModeContinue, false, nil
	}

	a := h.runner.agent
	if planJSON := plan.ExtractPlanJSON(response); planJSON != "" {
		if p, err := plan.ParsePlan(planJSON); err == nil && len(p.Steps) > 0 {
			yellow.Fprintf(a.output(), "📋 FC fallback: extracted %d-step plan from text. Switching to step-by-step...\n", len(p.Steps))
			h.runner.appendAssistantHistoryOnly(response)
			if err := a.runImplementationPhase(h.runner.ctx, p); err != nil {
				return normalModeContinue, true, err
			}
			a.runCompletionHooksWithRetry(h.runner.ctx)
			a.showTaskSummary()
			return normalModeDone, true, nil
		}
	}

	yellow.Fprintln(a.output(), "⚠️  Plan JSON detected but parse failed. Execute tools directly.")
	h.runner.appendAssistantHistoryOnly(response)
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: a.toolVisibilityPolicy(toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true}).normalModeRecoveryPrompt(normalModeRecoveryPromptDirectExecution),
	})
	return normalModeContinue, true, nil
}
