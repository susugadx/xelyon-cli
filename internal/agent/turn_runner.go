package agent

import (
	"context"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type TurnRunner struct {
	agent         *Agent
	ctx           context.Context
	lastToolCall  *tools.ToolCall
	sameCallCount int
}

type turnLoopDirective int

const (
	turnLoopProceed turnLoopDirective = iota
	turnLoopContinue
	turnLoopBreak
	turnLoopDone
	turnLoopReturn
)

type turnLoopPolicy struct {
	hardLimit        int
	requestResponse  func(iteration int) (string, error)
	onHardLimit      func(iteration int) (turnLoopDirective, error)
	afterPrepare     func(iteration int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error)
	onNoToolCalls    func(iteration int, response string) (turnLoopDirective, error)
	beforeToolCalls  func(iteration int, response string, toolCalls []*tools.ToolCall)
	executeToolCalls func(iteration int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error)
}

func newTurnRunner(agent *Agent, ctx context.Context) *TurnRunner {
	return &TurnRunner{agent: agent, ctx: ctx}
}

func (r *TurnRunner) promptManager() *PromptManager {
	return newPromptManager(r.agent)
}

func (r *TurnRunner) mutationTracker() *MutationTracker {
	return newMutationTracker(r.agent)
}

func (r *TurnRunner) resetLoopDetectionState() {
	r.lastToolCall = nil
	r.sameCallCount = 0
}

func (r *TurnRunner) prepareToolCalls(response string) []*tools.ToolCall {
	toolCalls := r.agent.parseToolCalls(response)
	for i, tc := range toolCalls {
		if tc.ID == "" {
			toolCalls[i].ID = fmt.Sprintf("call_rescue_%03d", i+1)
		}
	}
	return toolCalls
}

func (r *TurnRunner) buildLoopDetectFn() func(*tools.ToolCall) bool {
	return func(tc *tools.ToolCall) bool {
		cfg := r.agent.cfg()
		threshold := cfg.LoopDetection.Threshold
		if isSameToolCall(tc, r.lastToolCall) {
			r.sameCallCount++
			if r.sameCallCount >= threshold {
				yellow.Fprintf(r.agent.output(), "⚠️  Warning: Same tool call repeated %d times, stopping to prevent infinite loop\n", r.sameCallCount)
				yellow.Fprintf(r.agent.output(), "   Tool: %s\n", tc.Tool)
				return true
			}
		} else {
			r.sameCallCount = 1
		}
		r.lastToolCall = tc
		return false
	}
}

func (r *TurnRunner) appendDeferredLSPDiagnostics() {
	a := r.agent
	if diagMsg := r.mutationTracker().FlushDeferredDiagnostics(); diagMsg != "" && len(a.History) > 0 {
		a.History[len(a.History)-1].Content += diagMsg
	}
}

func (r *TurnRunner) executeToolCalls(
	response string,
	toolCalls []*tools.ToolCall,
	skipFn func(*tools.ToolCall) (bool, string),
	onResult ToolExecCallback,
) bool {
	if len(toolCalls) == 0 {
		return false
	}
	toolLoopDetected := r.agent.executeToolCallsWithParallel(
		r.ctx,
		toolCalls,
		r.buildLoopDetectFn(),
		skipFn,
		onResult,
	)
	r.appendDeferredLSPDiagnostics()
	return toolLoopDetected
}

func (r *TurnRunner) appendAssistantHistoryOnly(response string) {
	a := r.agent
	a.History = append(a.History, api.Message{
		Role:             "assistant",
		Content:          response,
		ReasoningContent: a.getLastReasoningContent(),
	})
}

func (r *TurnRunner) appendAssistantTurn(response string) {
	a := r.agent
	assistantMsg := api.Message{
		Role:             "assistant",
		Content:          response,
		ReasoningContent: a.getLastReasoningContent(),
	}
	a.History = append(a.History, assistantMsg)
	if a.session != nil {
		a.appendSessionMessageFromAPI(assistantMsg, a.CurrentModel)
	}
	if a.Stats != nil {
		a.Stats.AssistantMessages++
	}
}

func (r *TurnRunner) runTurnLoop(policy turnLoopPolicy) (turnLoopDirective, error) {
	for iteration := 0; ; iteration++ {
		if policy.hardLimit > 0 && iteration >= policy.hardLimit {
			if policy.onHardLimit != nil {
				return policy.onHardLimit(iteration)
			}
			return turnLoopBreak, nil
		}
		if policy.hardLimit == 0 {
			emitLoopWarning(r.agent, iteration)
		}

		response, err := policy.requestResponse(iteration)
		if err != nil {
			return turnLoopReturn, err
		}

		toolCalls := r.prepareToolCalls(response)
		if policy.afterPrepare != nil {
			directive, err := policy.afterPrepare(iteration, response, toolCalls)
			if err != nil {
				return turnLoopReturn, err
			}
			switch directive {
			case turnLoopContinue:
				continue
			case turnLoopBreak, turnLoopDone, turnLoopReturn:
				return directive, nil
			}
		}

		if len(toolCalls) == 0 {
			if policy.onNoToolCalls == nil {
				return turnLoopReturn, nil
			}
			directive, err := policy.onNoToolCalls(iteration, response)
			if err != nil {
				return turnLoopReturn, err
			}
			switch directive {
			case turnLoopContinue:
				continue
			case turnLoopBreak, turnLoopDone, turnLoopReturn:
				return directive, nil
			}
		}

		if policy.beforeToolCalls != nil {
			policy.beforeToolCalls(iteration, response, toolCalls)
		}

		if policy.executeToolCalls == nil {
			return turnLoopReturn, nil
		}
		directive, err := policy.executeToolCalls(iteration, response, toolCalls)
		if err != nil {
			return turnLoopReturn, err
		}
		switch directive {
		case turnLoopContinue:
			continue
		case turnLoopBreak, turnLoopDone, turnLoopReturn:
			return directive, nil
		}
	}
}

func (r *TurnRunner) RunNormalMode(input string, image *api.ImageData) error {
	return r.runNormalModeLoop(input, image)
}

func (r *TurnRunner) ExecuteStep(p *plan.Plan, step *plan.PlanStep, idx int, rs *retryState) error {
	return r.runPlanStepLoop(p, step, idx, rs)
}
