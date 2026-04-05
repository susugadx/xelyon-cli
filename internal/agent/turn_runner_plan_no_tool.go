package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type planStepNoToolHandler struct {
	runner *TurnRunner
	step   *plan.PlanStep
	state  *stepRunState
}

func newPlanStepNoToolHandler(r *TurnRunner, step *plan.PlanStep, state *stepRunState) *planStepNoToolHandler {
	return &planStepNoToolHandler{
		runner: r,
		step:   step,
		state:  state,
	}
}

func (h *planStepNoToolHandler) Handle(response string) (stepLoopAction, error) {
	if handled, action := h.handleAutoContinueQuestion(response); handled {
		return action, nil
	}
	if handled, action := h.handleAlreadyApplied(response); handled {
		return action, nil
	}
	if handled, action := h.handleWriteWithoutDiff(); handled {
		return action, nil
	}
	if handled, action := h.handleCompletionVerification(response); handled {
		return action, nil
	}

	h.completeStep(response)
	return stepLoopDone, nil
}

func (h *planStepNoToolHandler) handleAutoContinueQuestion(response string) (bool, stepLoopAction) {
	a := h.runner.agent
	if !isAIQuestionWithToolParser(response, a.parseToolCalls) || !h.canAutoContinue() {
		return false, stepLoopContinue
	}

	h.state.continueCount++
	yellow.Fprintf(a.output(), "⚠️  AI asked a question, auto-continuing (%d/%d)...\n", h.state.continueCount, h.maxContinues())
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: "[AUTO-CONTINUE] Yes, proceed with the step. Execute the required tools directly without asking for confirmation.",
	})
	return true, stepLoopContinue
}

func (h *planStepNoToolHandler) handleAlreadyApplied(response string) (bool, stepLoopAction) {
	a := h.runner.agent
	if !containsCompletionDeclaration(response) || h.state.beforeDiffHash == "" {
		return false, stepLoopContinue
	}

	afterDiffHash := getGitDiffHash()
	if afterDiffHash == "" || afterDiffHash == h.state.beforeDiffHash {
		return false, stepLoopContinue
	}

	a.displayAssistantResponse(prepareAssistantResponse(response))
	green.Fprintf(a.output(), "✓ Step %d completed (already applied)\n", h.step.ID)
	return true, stepLoopDone
}

func (h *planStepNoToolHandler) handleWriteWithoutDiff() (bool, stepLoopAction) {
	a := h.runner.agent
	if !h.state.stepHadWrites || h.state.stepHadNoChangeNeeded || !h.canAutoContinue() {
		return false, stepLoopContinue
	}

	afterDiffHash := getGitDiffHash()
	if h.state.beforeDiffHash == "" || afterDiffHash == "" || h.state.beforeDiffHash != afterDiffHash {
		return false, stepLoopContinue
	}

	h.state.continueCount++
	yellow.Fprintf(a.output(), "⚠️  Step %d: write tools executed but no file changes detected (%d/%d)\n",
		h.step.ID, h.state.continueCount, h.maxContinues())
	a.History = append(a.History, api.Message{
		Role: "user",
		Content: fmt.Sprintf("[SYSTEM] Step %d executed write tools but git diff shows no new changes. "+
			"The tool may have failed silently. Verify and retry.", h.step.ID),
	})
	return true, stepLoopContinue
}

func (h *planStepNoToolHandler) handleCompletionVerification(response string) (bool, stepLoopAction) {
	a := h.runner.agent
	if h.state.stepCompletionVerified {
		return false, stepLoopContinue
	}

	needsContinue, feedback := a.verifyCompletionWithDiagnostics(response)
	if !needsContinue {
		return false, stepLoopContinue
	}

	h.state.stepCompletionVerified = true
	yellow.Fprintln(a.output(), "⚠️  Step completion verification: LSP errors found in modified files")
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: feedback,
	})
	return true, stepLoopContinue
}

func (h *planStepNoToolHandler) completeStep(response string) {
	a := h.runner.agent
	a.displayAssistantResponse(prepareAssistantResponse(response))
	green.Fprintf(a.output(), "✓ Step %d completed\n", h.step.ID)
}

func (h *planStepNoToolHandler) maxContinues() int {
	return config.PlanMaxAutoContinues
}

func (h *planStepNoToolHandler) canAutoContinue() bool {
	return h.state.continueCount < h.maxContinues()
}
