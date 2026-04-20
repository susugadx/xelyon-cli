package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
)

type chatRequest struct {
	input      string
	image      *api.ImageData
	oneShot    bool
	startStats SessionStats
}

// chatCore は chat / ChatOnce の共有実装
// oneShot=true の場合: エラーを返し、対話向け後処理（usage表示・圧縮・context提案）をスキップ
func (a *Agent) chatCore(input string, image *api.ImageData, oneShot bool) error {
	req := &chatRequest{
		input:   input,
		image:   image,
		oneShot: oneShot,
	}

	a.prepareChatRequest(req)

	ctx, cleanup := a.beginChatRequestContext()
	defer cleanup()

	if !req.oneShot {
		a.checkTokenWarning()
	}

	if err := a.executeChatRequest(ctx, req); err != nil {
		return a.handleChatRequestError(req, err)
	}

	return a.finishChatRequest(req)
}

func (a *Agent) prepareChatRequest(req *chatRequest) {
	a.SetStatus(StateRunning, "Processing request", "処理中", "Wait for response", "応答を待ってください")
	a.resetSearchCodeTurnObservability()
	a.beginTaskTracking()

	req.input = a.AddGitHubHint(req.input)
	a.refreshProjectPrompt(req.input)

	if a.session != nil {
		a.appendSessionMessage("user", req.input, a.CurrentModel)
	}

	if a.Stats != nil {
		a.statsMu.Lock()
		a.Stats.UserMessages++
		req.startStats = *a.Stats
		a.statsMu.Unlock()
	}
}

func (a *Agent) beginTaskTracking() {
	prevChanges := len(a.changeStack) - a.taskChangeOffset
	a.taskChangeOffset = len(a.changeStack)

	a.taskTestResult = nil
	a.taskTestCommand = ""

	if prevChanges > 0 {
		yellow.Fprintf(a.output(), "⚠️  %d uncommitted changes from previous task\n", prevChanges)
		_, _ = fmt.Fprintln(a.output(), "💡 Run /commit or git commit to keep changes separate")
		_, _ = fmt.Fprintln(a.output())
	}
}

func (a *Agent) beginChatRequestContext() (context.Context, func()) {
	cfg := a.cfg()
	timeout := time.Duration(cfg.APIRetry.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	a.lastCancelReason = ""
	a.cancelFunc = cancel
	a.requestCtx = ctx
	a.debugCancelf("request started (timeout=%s, model=%s, provider=%s)", timeout, a.CurrentModel, a.ProviderName)

	cleanup := func() {
		a.debugCancelf("request finished (ctx_err=%v, cancel_reason=%q)", ctx.Err(), a.lastCancelReason)
		cancel()
		a.requestCtx = nil
		a.cancelFunc = nil
		a.lastCancelReason = ""
	}

	return ctx, cleanup
}

func (a *Agent) executeChatRequest(ctx context.Context, req *chatRequest) error {
	if req.oneShot {
		prevExcluded := a.registry().GetExcludedTools()
		defer a.registry().SetExcludedTools(prevExcluded)
	}

	toolVisibility := a.toolVisibilityPolicy(toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true})
	if a.PlanModeEnabled {
		toolVisibility = a.toolVisibilityPolicy(toolSurfacePhasePlan, toolVisibilityOptions{allowSubAgents: true})
		a.registry().SetExcludedTools(toolVisibility.excluded())
		return a.RunPlanMode(ctx, req.input)
	}

	a.registry().SetExcludedTools(toolVisibility.excluded())
	return a.runNormalChatRequest(ctx, req)
}

func (a *Agent) retryChatRequest(req *chatRequest) error {
	ctx := context.Background()
	if a.PlanModeEnabled {
		return a.RunPlanMode(ctx, req.input)
	}
	return a.runNormalChatRequest(ctx, req)
}

func (a *Agent) runNormalChatRequest(ctx context.Context, req *chatRequest) error {
	return a.runNormalMode(ctx, req.input, req.image)
}

func (a *Agent) handleChatRequestError(req *chatRequest, err error) error {
	if req.oneShot {
		a.ui().StopSpinner()
		return err
	}

	if errors.Is(err, context.Canceled) {
		yellow.Fprintln(a.output(), "\n⚠️  Response interrupted")
	} else if token.IsTokenLimitError(err) {
		retryFunc := func() error {
			return a.retryChatRequest(req)
		}
		if a.handleTokenLimitErrorWithRetry(err, retryFunc, a.PlanModeEnabled) {
			a.ui().StopSpinner()
			a.setReadyForInputStatus()
			return nil
		}
		red.Fprintf(a.output(), "Error: %v\n", err)
	} else {
		red.Fprintf(a.output(), "Error: %v\n", err)
	}

	a.ui().StopSpinner()
	reason := a.statusReasonForError(err)
	a.SetStatus(StateAborted, reason, reason, "Try again", "再試行してください")
	return nil
}

func (a *Agent) finishChatRequest(req *chatRequest) error {
	if req.oneShot {
		return nil
	}

	a.printTaskUsage(req.startStats)
	a.maybeAutoCompress()
	a.printContextSuggestion()
	a.setReadyForInputStatus()
	return nil
}

func (a *Agent) printContextSuggestion() {
	currentTokens := a.EstimateTokens()
	contextK := currentTokens / 1000

	baseTokens := a.EstimateSystemPromptTokens()
	if a.CurrentProvider != nil && a.CurrentProvider.IsFunctionCallingEnabled() {
		baseTokens += a.estimateToolDefinitionTokens()
	}

	if currentTokens <= baseTokens {
		dim.Fprintf(a.output(), "💡 Context %dK - clean state ✓\n", contextK)
		return
	}

	saved := currentTokens - baseTokens
	pricing := GetPricingInfo(a.ProviderName, a.CurrentModel)
	if pricing.InputCostPerM > 0 {
		savingPerTurn := float64(saved) / 1_000_000.0 * pricing.InputCostPerM * 0.5
		if savingPerTurn < 0.01 {
			dim.Fprintf(a.output(), "💡 Context %dK - clean state ✓\n", contextK)
			return
		}
		dim.Fprintf(a.output(), "💡 Context %dK - /clear saves ~$%.2f/turn, /compress keeps key context\n", contextK, savingPerTurn)
		return
	}

	dim.Fprintf(a.output(), "💡 Context %dK - /clear or /compress to reduce context\n", contextK)
}

func (a *Agent) setReadyForInputStatus() {
	a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
}
