package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
)

type planStepFailureHandler struct {
	runner *TurnRunner
	plan   *plan.Plan
	step   *plan.PlanStep
	idx    int
	retry  *retryState
	state  *stepRunState
}

func newPlanStepFailureHandler(r *TurnRunner, p *plan.Plan, step *plan.PlanStep, idx int, rs *retryState, state *stepRunState) *planStepFailureHandler {
	return &planStepFailureHandler{
		runner: r,
		plan:   p,
		step:   step,
		idx:    idx,
		retry:  rs,
		state:  state,
	}
}

func (h *planStepFailureHandler) Handle() (bool, error) {
	if h == nil || h.state == nil || h.state.lastFailedResult == "" {
		return false, nil
	}

	level := h.retry.recordFailure(h.state.lastFailedResult)
	switch level {
	case stalledNone:
		return true, h.retryCurrentStep()
	case stalledSoft:
		return true, h.retryCurrentStepWithStrategyChange()
	default:
		return h.handleSelectorAction()
	}
}

func (h *planStepFailureHandler) retryCurrentStep() error {
	a := h.runner.agent
	a.ui().StopSpinner()
	red.Fprintf(a.output(), "❌ Step %d Failed (auto-retry %d)\n", h.step.ID, h.retry.count)
	yellow.Fprintf(a.output(), "🔄 Retrying...\n")

	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: buildPlanStepRetryMessage(h.state.lastFailedResult, h.retry.count),
	})
	return h.runner.ExecuteStep(h.plan, h.step, h.idx, h.retry)
}

func (h *planStepFailureHandler) retryCurrentStepWithStrategyChange() error {
	a := h.runner.agent
	a.ui().StopSpinner()
	yellow.Fprintf(a.output(), "⚠️  Step %d: similar failure repeated %d times (auto-retry %d)\n", h.step.ID, h.retry.sameCount, h.retry.count)
	yellow.Fprintf(a.output(), "🔄 Retrying with strategy change...\n")

	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: buildPlanStepStrategyChangeMessage(h.state.lastFailedResult, h.retry.sameCount, h.retry.count),
	})
	return h.runner.ExecuteStep(h.plan, h.step, h.idx, h.retry)
}

func (h *planStepFailureHandler) handleSelectorAction() (bool, error) {
	a := h.runner.agent
	a.SetStatus(StateWaitingApproval, "Step failed - waiting for action", "ステップ失敗 - アクション待ち", "Choose action", "アクションを選択")
	a.ui().StopSpinner()

	for {
		action, comment := promptFailureActionWithSelector(a.ui().PromptIO(), h.step, h.state.lastFailedResult, h.state.lastFailReason, h.retry.count)
		switch action {
		case plan.FailureActionRetry:
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: buildPlanStepManualRetryMessage(h.state.lastFailedResult),
			})
			return true, h.runner.ExecuteStep(h.plan, h.step, h.idx, &retryState{})
		case plan.FailureActionComment:
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: buildPlanStepCommentRetryMessage(comment, h.state.lastFailedResult),
			})
			return true, h.runner.ExecuteStep(h.plan, h.step, h.idx, &retryState{})
		case plan.FailureActionSkip:
			yellow.Fprintf(a.output(), "⏭️  Step %d skipped by user\n", h.step.ID)
			return true, nil
		case plan.FailureActionAbort:
			red.Fprintf(a.output(), "🛑 Step %d aborted by user\n", h.step.ID)
			return true, fmt.Errorf("step %d aborted by user: %s", h.step.ID, h.state.lastFailReason)
		}
	}
}

func buildPlanStepRetryMessage(lastFailedResult string, attempt int) string {
	return fmt.Sprintf("The previous step FAILED with the following error:\n\n%s\n\n%s",
		lastFailedResult, planModeRetryInstruction(attempt))
}

func buildPlanStepStrategyChangeMessage(lastFailedResult string, sameCount, attempt int) string {
	return fmt.Sprintf("The previous step FAILED with the following error:\n\n%s\n\n"+
		"WARNING: A similar failure has now occurred %d times in a row.\n"+
		"Your previous approach is likely wrong — do not repeat the same fix pattern.\n\n%s",
		lastFailedResult, sameCount, planModeRetryInstruction(attempt))
}

func buildPlanStepManualRetryMessage(lastFailedResult string) string {
	return fmt.Sprintf(`The previous step FAILED with the following error:

%s

Please:
1. Analyze the error carefully
2. Identify the root cause
3. Fix the code or configuration
4. Re-run the step to verify the fix

Do NOT skip this step. The issue must be resolved before proceeding.`, lastFailedResult)
}

func buildPlanStepCommentRetryMessage(comment, lastFailedResult string) string {
	return fmt.Sprintf(`The previous step FAILED. Here are the user's instructions for fixing it:

%s

Error that occurred:
%s

Please follow these instructions to fix the issue and retry the step.`, comment, lastFailedResult)
}
