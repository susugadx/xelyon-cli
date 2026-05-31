package agent

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/finalcheck"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	promptnormal "github.com/susugadx/xelyon-cli/internal/prompt/normal"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/turnsupport"
)

const (
	maxTextPlanRedirects = 2
	maxTextPlanHardLimit = 5
)

type normalModeState struct {
	rs                    turnsupport.RetryState
	finalCheckRetry       finalcheck.RetryState
	textPlanRedirectCount int
	fallbackResponse      string
	reachedHardLimit      bool
	turnMutations         turnsupport.MutationState
}

type normalModeAction int

const (
	normalModeContinue normalModeAction = iota
	normalModeBreak
	normalModeDone
)

func (r *TurnRunner) requestNormalModeResponse(input string, image *api.ImageData, iteration int) (string, error) {
	a := r.agent
	if iteration == 0 {
		r.promptManager().RefreshProjectPromptIfDirty(input)
	}
	effectivePrompt := prompt.StripPlanningReferences(a.SystemPrompt)

	requestCtx := a.requestContext(r.ctx)
	if iteration == 0 && image != nil {
		requestCtx, history := a.providerFacingHistoryExcludingLatestMessageForRequest(requestCtx)
		response, err := a.CurrentProvider.ChatWithImage(
			requestCtx, effectivePrompt, history, input+promptnormal.NormalModePrompt, image, a.CurrentModel,
		)
		if err != nil {
			a.ui().StopSpinner()
			return "", fmt.Errorf("API call failed: %w", err)
		}
		return response, nil
	}

	requestCtx, history := a.providerFacingHistoryForRequest(requestCtx)
	response, err := a.CurrentProvider.ChatWithTools(
		requestCtx,
		effectivePrompt,
		history,
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

func (r *TurnRunner) processNormalModeToolCalls(response string, toolCalls []*tools.ToolCall, state *normalModeState, rs *turnsupport.RetryState) error {
	a := r.agent
	handler := newNormalModeToolResultHandler(r, state)

	if len(toolCalls) > 0 {
		a.addToolCallsToHistory(response, toolCalls)
	}

	toolLoopDetected := r.executeToolCalls(response, toolCalls, nil, func(_ int, tc *tools.ToolCall, result toolruntime.Result) {
		handler.Handle(tc, result)
	})
	if toolLoopDetected {
		return fmt.Errorf("tool loop detected")
	}
	return newNormalModeFailureHandler(r, rs, handler.LastFailedResult()).Handle()
}
