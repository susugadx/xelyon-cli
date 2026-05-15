package agent

import (
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	promptnormal "github.com/susugadx/xelyon-cli/internal/prompt/normal"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// normalModeTurnOrchestrator は通常ターンのループ方針を組み立てる。
// 実処理は既存ハンドラーへ委譲し、ここでは制御フローのみを管理する。
type normalModeTurnOrchestrator struct {
	runner *TurnRunner
	input  string
	image  *api.ImageData

	cfg             *config.Config
	hardLimit       int
	state           *normalModeState
	planningHandler *normalModePlanningHandler
}

func newNormalModeTurnOrchestrator(r *TurnRunner, input string, image *api.ImageData) *normalModeTurnOrchestrator {
	cfg := r.agent.cfg()
	return &normalModeTurnOrchestrator{
		runner:          r,
		input:           input,
		image:           image,
		cfg:             cfg,
		hardLimit:       normalizeToolLoopLimit(cfg.General.ToolLoopLimit),
		state:           &normalModeState{turnMutations: newTurnMutationState()},
		planningHandler: newNormalModePlanningHandler(r),
	}
}

func (o *normalModeTurnOrchestrator) Run() error {
	o.appendNormalModeInputToHistory()

	directive, err := o.runner.runTurnLoop(turnLoopPolicy{
		hardLimit:        o.hardLimit,
		onHardLimit:      o.onHardLimit,
		requestResponse:  o.requestResponse,
		afterPrepare:     o.afterPrepare,
		onNoToolCalls:    o.onNoToolCalls,
		beforeToolCalls:  o.beforeToolCalls,
		executeToolCalls: o.executeToolCalls,
		afterToolResults: o.afterToolResults,
	})
	if err != nil {
		return err
	}
	return o.handleDirective(directive)
}

func (o *normalModeTurnOrchestrator) appendNormalModeInputToHistory() {
	o.runner.agent.History = append(o.runner.agent.History, api.Message{
		Role:    "user",
		Content: o.input + promptnormal.NormalModePrompt,
	})
}

func (o *normalModeTurnOrchestrator) onHardLimit(_ int) (turnLoopDirective, error) {
	o.state.reachedHardLimit = true
	return turnLoopBreak, nil
}

func (o *normalModeTurnOrchestrator) requestResponse(iteration int) (string, error) {
	return o.runner.requestNormalModeResponse(o.input, o.image, iteration)
}

func (o *normalModeTurnOrchestrator) afterPrepare(_ int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error) {
	action, handled, err := o.planningHandler.HandlePlanJSONFallback(response, toolCalls)
	if err != nil {
		return turnLoopReturn, err
	}
	if handled {
		if action == normalModeDone {
			return turnLoopReturn, nil
		}
		return turnLoopContinue, nil
	}

	o.runner.debugLogToolCalls(response, toolCalls)
	return turnLoopProceed, nil
}

func (o *normalModeTurnOrchestrator) onNoToolCalls(_ int, response string) (turnLoopDirective, error) {
	switch o.runner.handleNormalModeNoToolResponse(response, o.cfg, o.state) {
	case normalModeContinue:
		return turnLoopContinue, nil
	case normalModeBreak:
		return turnLoopBreak, nil
	case normalModeDone:
		return turnLoopDone, nil
	default:
		return turnLoopProceed, nil
	}
}

func (o *normalModeTurnOrchestrator) beforeToolCalls(_ int, response string, toolCalls []*tools.ToolCall) {
	o.runner.agent.maybePrintAssistantPhaseUpdate(response, toolCalls)
}

func (o *normalModeTurnOrchestrator) executeToolCalls(_ int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error) {
	if err := o.runner.processNormalModeToolCalls(response, toolCalls, o.state, &o.state.rs); err != nil {
		return turnLoopReturn, err
	}
	return turnLoopProceed, nil
}

func (o *normalModeTurnOrchestrator) afterToolResults(_ int, _ string, _ []*tools.ToolCall) (turnLoopDirective, error) {
	return turnLoopProceed, nil
}

func (o *normalModeTurnOrchestrator) handleDirective(directive turnLoopDirective) error {
	a := o.runner.agent
	switch directive {
	case turnLoopBreak:
		if o.state.reachedHardLimit {
			yellow.Fprintf(a.output(), "⚠️  Tool loop limit reached (%d iterations)\n", o.hardLimit)
		}
		if o.state.fallbackResponse != "" {
			a.showAssistantResponse(o.state.fallbackResponse)
		}
		a.showTaskSummary()
	case turnLoopDone:
		a.showTaskSummary()
	}
	return nil
}

func (r *TurnRunner) runNormalModeLoop(input string, image *api.ImageData) error {
	return newNormalModeTurnOrchestrator(r, input, image).Run()
}
