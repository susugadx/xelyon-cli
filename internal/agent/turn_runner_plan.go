package agent

import (
	"fmt"

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
	directive, err := r.runTurnLoop(turnLoopPolicy{
		hardLimit: hardLimit,
		onHardLimit: func(_ int) (turnLoopDirective, error) {
			return turnLoopReturn, fmt.Errorf("step %d exceeded max iterations (%d)", step.ID, hardLimit)
		},
		requestResponse: func(_ int) (string, error) {
			response, err := r.requestPlanStepResponse(stepPrompt)
			if err != nil {
				return "", fmt.Errorf("step %d failed: %w", step.ID, err)
			}
			return response, nil
		},
		afterPrepare: func(_ int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error) {
			if len(toolCalls) > 0 {
				a.addToolCallsToHistory(response, toolCalls)
			} else {
				r.appendAssistantTurn(response)
			}
			return turnLoopProceed, nil
		},
		onNoToolCalls: func(_ int, response string) (turnLoopDirective, error) {
			action, err := r.handleStepNoToolResponse(response, step, state)
			if err != nil {
				return turnLoopReturn, err
			}
			switch action {
			case stepLoopContinue:
				return turnLoopContinue, nil
			case stepLoopDone:
				return turnLoopDone, nil
			default:
				return turnLoopProceed, nil
			}
		},
		executeToolCalls: func(_ int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error) {
			handled, err := r.processStepToolCalls(toolCalls, step, p, idx, rs, state)
			if err != nil {
				return turnLoopReturn, err
			}
			if handled {
				return turnLoopReturn, nil
			}
			return turnLoopProceed, nil
		},
	})
	if err != nil {
		return err
	}

	if directive == turnLoopDone || directive == turnLoopReturn {
		return nil
	}
	return nil
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
	handler := newPlanStepToolResultHandler(r, state)

	r.executeToolCalls("", execToolCalls, nil, func(_ int, toolCall *tools.ToolCall, result string, change *tools.FileChange) {
		handler.Handle(toolCall, result, change)
	})

	if state.lastFailedResult == "" {
		return false, nil
	}

	return newPlanStepFailureHandler(r, p, step, idx, rs, state).Handle()
}
