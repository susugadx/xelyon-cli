package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/token"
)

type chatRequest struct {
	input           string
	image           *api.ImageData
	oneShot         bool
	startStats      SessionStats
	autoCompression *autoCompressionTurnState
}

// chatCore は chat / ChatOnce の共有実装
// oneShot=true の場合: エラーを返し、対話向け後処理（usage表示・圧縮・context提案）をスキップ
func (a *Agent) chatCore(input string, image *api.ImageData, oneShot bool) error {
	req := newChatRequest(input, image, oneShot)

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

func newChatRequest(input string, image *api.ImageData, oneShot bool) *chatRequest {
	req := &chatRequest{
		input:   input,
		image:   image,
		oneShot: oneShot,
	}
	if !oneShot {
		req.autoCompression = newAutoCompressionTurnState()
	}
	return req
}

func (a *Agent) prepareChatRequest(req *chatRequest) {
	a.SetStatus(StateRunning, "Processing request", "処理中", "Wait for response", "応答を待ってください")
	a.resetSearchCodeTurnObservability()
	a.beginTaskTracking()

	a.refreshProjectPrompt(req.input)
	a.clearResponseContextForActiveContextRequest()

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
	a.taskPlanVerification = nil

	if prevChanges > 0 {
		yellow.Fprintf(a.output(), "⚠️  %d uncommitted changes from previous task\n", prevChanges)
		_, _ = fmt.Fprintln(a.output(), "💡 Run /commit or git commit to keep changes separate")
		_, _ = fmt.Fprintln(a.output())
	}
}

func (a *Agent) executeChatRequest(ctx context.Context, req *chatRequest) error {
	if req.oneShot {
		prevExcluded := a.registry().GetExcludedTools()
		defer a.registry().SetExcludedTools(prevExcluded)
	}

	return a.runChatRequestForCurrentMode(ctx, req)
}

func (a *Agent) runChatRequestForCurrentMode(ctx context.Context, req *chatRequest) error {
	if a.PlanModeEnabled {
		return a.runPlanModeChatRequest(ctx, req)
	}

	a.applyChatRequestToolVisibility(toolSurfacePhaseNormal)
	return a.runNormalChatRequest(ctx, req)
}

func (a *Agent) runPlanModeChatRequest(ctx context.Context, req *chatRequest) error {
	a.applyChatRequestToolVisibility(toolSurfacePhasePlan)
	handoff, err := a.runPlanModeWithAutoCompression(ctx, req.input, req.autoCompression)
	if err != nil || handoff == nil {
		return err
	}
	return a.executeApprovedPlanHandoff(ctx, req, handoff)
}

func (a *Agent) applyChatRequestToolVisibility(phase toolSurfacePhase) {
	toolVisibility := a.toolVisibilityPolicy(phase, toolVisibilityOptions{allowSubAgents: true})
	previousSurface, surfaceChanged := a.refreshMCPToolSurface()
	if surfaceChanged {
		a.configureCurrentProviderMCPTools()
		a.rebuildSystemPromptForCurrentProvider()
	}
	a.registry().SetExcludedTools(mergeSurfaceManagedExcludedToolsWithRuntimeExclusions(
		a.registry().GetExcludedTools(),
		toolVisibility,
		previousSurface.omittedExportedNames(),
		a.currentMCPBudgetExcludedToolNames(),
	))
}

func (a *Agent) executeApprovedPlanHandoff(ctx context.Context, req *chatRequest, handoff *plan.ImplementationHandoff) error {
	input := strings.TrimSpace(handoff.NormalModeInput())
	if input == "" {
		return nil
	}

	a.setTaskPlanVerification(handoff.VerificationHints())
	req.input = input
	if a.session != nil {
		a.appendSessionMessage("user", req.input, a.CurrentModel)
	}

	a.applyChatRequestToolVisibility(toolSurfacePhaseNormal)
	a.SetStatus(StateRunning, "Implementing approved plan", "承認済み計画を実装中", "Wait for implementation", "実装完了を待ってください")
	cyan.Fprintln(a.output(), "\nStarting implementation from approved plan...")
	return a.runNormalChatRequest(ctx, req)
}

func (a *Agent) retryChatRequest(req *chatRequest) error {
	ctx := context.Background()
	return a.runChatRequestForCurrentMode(ctx, req)
}

func (a *Agent) runNormalChatRequest(ctx context.Context, req *chatRequest) error {
	if req.oneShot {
		return a.runNormalMode(ctx, req.input, req.image)
	}
	return a.runNormalModeWithAutoCompression(ctx, req.input, req.image, req.autoCompression)
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
		if a.handleTokenLimitErrorWithRetry(err, retryFunc) {
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
	a.maybeAutoCompressAfterTurn(req.autoCompression)
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
	cfg := a.cfg()
	pricing := cost.GetPricingInfoForConfig(cfg, a.activeModelProviderConfigKey(cfg), a.CurrentModel)
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
	a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or / for commands", "リクエスト、または / でコマンド候補を入力")
}
