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
	if len(toolCalls) != 0 {
		return normalModeContinue, false, nil
	}

	a := h.runner.agent
	planJSON := plan.ExtractPlanJSONForNormalModeRecovery(response)
	if planJSON == "" {
		return normalModeContinue, false, nil
	}

	if _, err := plan.ParsePlan(planJSON); err == nil {
		yellow.Fprintln(a.output(), "⚠️  Plan JSON detected in normal mode. Execute tools directly instead.")
	} else {
		yellow.Fprintln(a.output(), "⚠️  Plan JSON detected but parse failed. Execute tools directly.")
	}
	h.runner.appendAssistantHistoryOnly(response)
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: a.toolVisibilityPolicy(toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true}).normalModeRecoveryPrompt(normalModeRecoveryPromptDirectExecution),
	})
	return normalModeContinue, true, nil
}
