package agent

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	promptnormal "github.com/susugadx/xelyon-cli/internal/prompt/normal"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

const (
	maxTextPlanRedirects = 2
	maxTextPlanHardLimit = 5
)

type normalModeState struct {
	rs                    retryState
	finalCheckRetry       finalCheckRetryState
	textPlanRedirectCount int
	fallbackResponse      string
	reachedHardLimit      bool
	turnMutations         turnMutationState
}

type normalModeAction int

const (
	normalModeContinue normalModeAction = iota
	normalModeBreak
	normalModeDone
)

func (r *TurnRunner) runNormalModeLoop(input string, image *api.ImageData) error {
	a := r.agent
	planningHandler := newNormalModePlanningHandler(r)

	normalModeInput := input + promptnormal.NormalModePrompt
	a.History = append(a.History, api.Message{Role: "user", Content: normalModeInput})

	cfg := a.cfg()
	hardLimit := normalizeToolLoopLimit(cfg.General.ToolLoopLimit)
	state := &normalModeState{
		turnMutations: newTurnMutationState(),
	}
	directive, err := r.runTurnLoop(turnLoopPolicy{
		hardLimit: hardLimit,
		onHardLimit: func(_ int) (turnLoopDirective, error) {
			state.reachedHardLimit = true
			return turnLoopBreak, nil
		},
		requestResponse: func(iteration int) (string, error) {
			return r.requestNormalModeResponse(input, image, iteration)
		},
		afterPrepare: func(_ int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error) {
			action, handled, err := planningHandler.HandlePlanJSONFallback(response, toolCalls)
			if err != nil {
				return turnLoopReturn, err
			}
			if handled {
				if action == normalModeDone {
					return turnLoopReturn, nil
				}
				return turnLoopContinue, nil
			}

			r.debugLogToolCalls(response, toolCalls)
			return turnLoopProceed, nil
		},
		onNoToolCalls: func(_ int, response string) (turnLoopDirective, error) {
			switch r.handleNormalModeNoToolResponse(response, cfg, state) {
			case normalModeContinue:
				return turnLoopContinue, nil
			case normalModeBreak:
				return turnLoopBreak, nil
			case normalModeDone:
				return turnLoopDone, nil
			default:
				return turnLoopProceed, nil
			}
		},
		beforeToolCalls: func(_ int, response string, toolCalls []*tools.ToolCall) {
			a.maybePrintAssistantPhaseUpdate(response, toolCalls)
		},
		executeToolCalls: func(_ int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error) {
			if err := r.processNormalModeToolCalls(response, toolCalls, state, &state.rs); err != nil {
				return turnLoopReturn, err
			}
			return turnLoopProceed, nil
		},
	})
	if err != nil {
		return err
	}

	switch directive {
	case turnLoopBreak:
		if state.reachedHardLimit {
			yellow.Fprintf(a.output(), "⚠️  Tool loop limit reached (%d iterations)\n", hardLimit)
		}
		if state.fallbackResponse != "" {
			a.showAssistantResponse(state.fallbackResponse)
		}
		a.showTaskSummary()
		return nil
	case turnLoopDone:
		a.showTaskSummary()
		return nil
	default:
		return nil
	}
}

func (r *TurnRunner) requestNormalModeResponse(input string, image *api.ImageData, iteration int) (string, error) {
	a := r.agent
	if iteration == 0 {
		r.promptManager().RefreshProjectPromptIfDirty(input)
	}
	effectivePrompt := prompt.StripPlanningReferences(a.SystemPrompt)

	requestCtx := a.requestContext(r.ctx)
	if iteration == 0 && image != nil {
		response, err := a.CurrentProvider.ChatWithImage(
			requestCtx, effectivePrompt, a.History[:len(a.History)-1], input+promptnormal.NormalModePrompt, image, a.CurrentModel,
		)
		if err != nil {
			a.ui().StopSpinner()
			return "", fmt.Errorf("API call failed: %w", err)
		}
		return response, nil
	}

	response, err := a.CurrentProvider.ChatWithTools(
		requestCtx,
		effectivePrompt,
		a.History,
		a.CurrentModel,
	)
	if tc, ok := a.CurrentProvider.(interface{ ClearToolChoice() }); ok {
		tc.ClearToolChoice()
	}
	if err != nil {
		a.ui().StopSpinner()
		return "", fmt.Errorf("API call failed: %w", err)
	}
	return response, nil
}

func (r *TurnRunner) debugLogToolCalls(response string, toolCalls []*tools.ToolCall) {
	a := r.agent
	if os.Getenv("XELYON_DEBUG_TOOLS") != "1" {
		return
	}

	_, _ = fmt.Fprintf(a.errorOutput(), "[DEBUG Tools] Response length: %d, ToolCalls found: %d\n", len(response), len(toolCalls))
	if len(response) < config.DebugPreviewLen {
		_, _ = fmt.Fprintf(a.errorOutput(), "[DEBUG Tools] Response: %s\n", response)
	} else {
		_, _ = fmt.Fprintf(a.errorOutput(), "[DEBUG Tools] Response (first %d): %s...\n", config.DebugPreviewLen, response[:config.DebugPreviewLen])
	}
	for i, tc := range toolCalls {
		_, _ = fmt.Fprintf(a.errorOutput(), "[DEBUG Tools] ToolCall[%d]: tool=%s, args=%v\n", i, tc.Tool, tc.Args)
	}
}

func (r *TurnRunner) handleNormalModeNoToolResponse(response string, cfg *config.Config, state *normalModeState) normalModeAction {
	return newNormalModeNoToolHandler(r, cfg, state).Handle(response)
}

func (r *TurnRunner) processNormalModeToolCalls(response string, toolCalls []*tools.ToolCall, state *normalModeState, rs *retryState) error {
	a := r.agent
	handler := newNormalModeToolResultHandler(r, state)

	if len(toolCalls) > 0 {
		a.addToolCallsToHistory(response, toolCalls)
	}

	toolLoopDetected := r.executeToolCalls(response, toolCalls, nil, func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		handler.Handle(tc, result, change)
	})
	if toolLoopDetected {
		return fmt.Errorf("tool loop detected")
	}
	return newNormalModeFailureHandler(r, rs, handler.LastFailedResult()).Handle()
}
