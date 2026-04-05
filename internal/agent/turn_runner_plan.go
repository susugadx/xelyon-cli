package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type stepRunState struct {
	continueCount          int
	lastFailedResult       string
	lastFailReason         string
	stepHadWrites          bool
	stepHadNoChangeNeeded  bool
	beforeDiffHash         string
	stepCompletionVerified bool
}

type stepLoopAction int

const (
	stepLoopContinue stepLoopAction = iota
	stepLoopDone
)

func (r *TurnRunner) runPlanStepLoop(p *plan.Plan, step *plan.PlanStep, idx int, rs *retryState) error {
	a := r.agent

	// Each step attempt gets a fresh loop-detection budget.
	r.resetLoopDetectionState()

	if rs.count > 0 {
		a.ui().StopSpinner()
		yellow.Fprintf(a.output(), "🔄 Retry attempt %d for step %d...\n", rs.count, step.ID)
	}

	stepPrompt := promptplan.BuildStepPrompt(step.ID, step.Description, step.Tools)
	if rs.count == 0 {
		a.History = append(a.History, api.Message{Role: "user", Content: stepPrompt})
	}

	cfg := a.cfg()
	hardLimit := normalizeToolLoopLimit(cfg.General.ToolLoopLimit)
	state := &stepRunState{
		beforeDiffHash: getGitDiffHash(),
	}

	for j := 0; ; j++ {
		if hardLimit > 0 && j >= hardLimit {
			return fmt.Errorf("step %d exceeded max iterations (%d)", step.ID, hardLimit)
		}
		if hardLimit == 0 {
			emitLoopWarning(a, j)
		}

		response, err := r.requestPlanStepResponse(stepPrompt)
		if err != nil {
			return fmt.Errorf("step %d failed: %w", step.ID, err)
		}

		execToolCalls := r.prepareToolCalls(response)
		if len(execToolCalls) > 0 {
			a.addToolCallsToHistory(response, execToolCalls)
		} else {
			r.appendAssistantTurn(response)
		}

		if len(execToolCalls) == 0 {
			action, err := r.handleStepNoToolResponse(response, step, state)
			if err != nil {
				return err
			}
			if action == stepLoopDone {
				return nil
			}
			continue
		}

		handled, err := r.processStepToolCalls(execToolCalls, step, p, idx, rs, state)
		if err != nil || handled {
			return err
		}
	}
}

func (r *TurnRunner) requestPlanStepResponse(stepPrompt string) (string, error) {
	a := r.agent
	r.promptManager().RefreshProjectPromptIfDirty(stepPrompt)

	response, err := a.CurrentProvider.ChatWithTools(
		a.requestContext(r.ctx),
		a.SystemPrompt,
		a.History,
		a.CurrentModel,
	)
	if err != nil {
		a.ui().StopSpinner()
		return "", err
	}
	return response, nil
}

func (r *TurnRunner) handleStepNoToolResponse(response string, step *plan.PlanStep, state *stepRunState) (stepLoopAction, error) {
	return newPlanStepNoToolHandler(r, step, state).Handle(response)
}

func (r *TurnRunner) processStepToolCalls(execToolCalls []*tools.ToolCall, step *plan.PlanStep, p *plan.Plan, idx int, rs *retryState, state *stepRunState) (bool, error) {
	a := r.agent
	tracker := r.mutationTracker()

	r.executeToolCalls("", execToolCalls, r.planStepSkipFn, func(_ int, toolCall *tools.ToolCall, result string, change *tools.FileChange) {
		a.appendSessionToolExecution(toolCall, result)
		tracker.RecordToolResult(toolCall, result, change)

		if tools.IsWriteTool(toolCall.Tool) {
			state.stepHadWrites = true
			if strings.Contains(result, "no files found") ||
				strings.Contains(result, "Total matches: 0") ||
				strings.Contains(result, "no change needed") {
				state.stepHadNoChangeNeeded = true
			}
		}

		if toolCall.Tool == "bash" || tools.IsWriteTool(toolCall.Tool) {
			if failed, reason := plan.ContainsFailure(result); failed {
				state.lastFailedResult = result
				state.lastFailReason = reason
			}
		}

		a.appendToolResultToHistory(toolCall, result)
	})

	if state.lastFailedResult == "" {
		return false, nil
	}

	return newPlanStepFailureHandler(r, p, step, idx, rs, state).Handle()
}

func (r *TurnRunner) planStepSkipFn(tc *tools.ToolCall) (bool, string) {
	if tc.Tool == "create_plan" || tc.Tool == "update_plan" {
		yellow.Fprintf(r.agent.output(), "⚠️  Ignored deprecated planning tool call: %s\n", tc.Tool)
		return true, fmt.Sprintf("[%s] Ignored: planning tools are deprecated. Continue with current step.", tc.Tool)
	}
	return false, ""
}
