package agent

import (
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type normalModeNoToolHandler struct {
	runner *TurnRunner
	cfg    *config.Config
	state  *normalModeState
}

func newNormalModeNoToolHandler(r *TurnRunner, cfg *config.Config, state *normalModeState) *normalModeNoToolHandler {
	return &normalModeNoToolHandler{
		runner: r,
		cfg:    cfg,
		state:  state,
	}
}

func (h *normalModeNoToolHandler) Handle(response string) normalModeAction {
	if handled, action := h.handleTextPlanRedirect(response); handled {
		return action
	}
	if handled, action := h.handleCompletionVerification(response); handled {
		return action
	}
	if handled, action := h.handleCompletionHooks(response); handled {
		return action
	}

	h.runner.agent.handleNormalResponse(response)
	return normalModeDone
}

func (h *normalModeNoToolHandler) handleTextPlanRedirect(response string) (bool, normalModeAction) {
	a := h.runner.agent
	steps := extractTextPlan(response)
	if containsCompletionDeclaration(response) || len(steps) < 5 || !isActionPlan(steps) {
		return false, normalModeContinue
	}

	h.state.textPlanRedirectCount++
	if h.state.textPlanRedirectCount > maxTextPlanHardLimit {
		yellow.Fprintf(a.output(), "⚠️  Text plan detected %d times without tool use. Returning response to user.\n", h.state.textPlanRedirectCount)
		h.runner.appendAssistantHistoryOnly(response)
		h.state.fallbackResponse = response
		return true, normalModeBreak
	}

	if h.state.textPlanRedirectCount > maxTextPlanRedirects {
		yellow.Fprintf(a.output(), "⚠️  Text plan detected %d times. Forcing direct execution.\n", h.state.textPlanRedirectCount)
		h.runner.appendAssistantHistoryOnly(response)
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: a.toolVisibilityPolicy(toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true}).normalModeRecoveryPrompt(normalModeRecoveryPromptStopPlanning),
		})
		return true, normalModeContinue
	}

	yellow.Fprintf(a.output(), "⚠️  Text plan detected (%d steps). Execute tools directly instead. (%d/%d)\n",
		len(steps), h.state.textPlanRedirectCount, maxTextPlanRedirects)
	h.runner.appendAssistantHistoryOnly(response)
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: a.toolVisibilityPolicy(toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true}).normalModeRecoveryPrompt(normalModeRecoveryPromptNoTextPlan),
	})
	return true, normalModeContinue
}

func (h *normalModeNoToolHandler) handleCompletionVerification(response string) (bool, normalModeAction) {
	a := h.runner.agent
	if h.state.completionVerified {
		return false, normalModeContinue
	}

	needsContinue, feedback := a.verifyCompletionWithDiagnostics(response)
	if !needsContinue {
		return false, normalModeContinue
	}

	h.state.completionVerified = true
	yellow.Fprintln(a.output(), "⚠️  Completion verification: LSP errors found in modified files")
	h.runner.appendAssistantHistoryOnly(response)
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: feedback,
	})
	return true, normalModeContinue
}

func (h *normalModeNoToolHandler) handleCompletionHooks(response string) (bool, normalModeAction) {
	a := h.runner.agent
	if !containsCompletionDeclaration(response) || len(h.cfg.Hooks.OnCompletion) == 0 {
		return false, normalModeContinue
	}

	changedFiles := a.getTaskChangedFiles()
	if len(changedFiles) == 0 {
		return false, normalModeContinue
	}

	hookNeedsContinue, hookFeedback := a.checkGitDiffEmpty()
	if !hookNeedsContinue {
		hookNeedsContinue, hookFeedback = a.runCompletionHooks(changedFiles)
	}
	if !hookNeedsContinue {
		return false, normalModeContinue
	}

	h.state.hookRetryCount++
	maxRetry := h.cfg.Hooks.MaxRetry
	if maxRetry <= 0 {
		maxRetry = 3
	}
	if h.state.hookRetryCount >= maxRetry {
		yellow.Fprintf(a.output(), "⚠️  Hook retry limit reached (%d/%d). Proceeding with completion.\n", h.state.hookRetryCount, maxRetry)
		return false, normalModeContinue
	}

	yellow.Fprintf(a.output(), "⚠️  Completion hook failed (%d/%d). Asking AI to fix...\n", h.state.hookRetryCount, maxRetry)
	h.runner.appendAssistantHistoryOnly(response)
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: hookFeedback,
	})
	return true, normalModeContinue
}
