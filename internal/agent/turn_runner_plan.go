package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
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
	a.refreshProjectPromptIfDirty(stepPrompt)

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
	a := r.agent
	maxContinues := config.PlanMaxAutoContinues

	if isAIQuestionWithToolParser(response, a.parseToolCalls) && state.continueCount < maxContinues {
		state.continueCount++
		yellow.Fprintf(a.output(), "⚠️  AI asked a question, auto-continuing (%d/%d)...\n", state.continueCount, maxContinues)
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: "[AUTO-CONTINUE] Yes, proceed with the step. Execute the required tools directly without asking for confirmation.",
		})
		return stepLoopContinue, nil
	}

	if containsCompletionDeclaration(response) && state.beforeDiffHash != "" {
		afterDiffHash := getGitDiffHash()
		if afterDiffHash != "" && afterDiffHash != state.beforeDiffHash {
			a.printFinalAssistantResponse(response)
			green.Fprintf(a.output(), "✓ Step %d completed (already applied)\n", step.ID)
			return stepLoopDone, nil
		}
	}

	if state.stepHadWrites && !state.stepHadNoChangeNeeded {
		afterDiffHash := getGitDiffHash()
		if state.beforeDiffHash != "" && afterDiffHash != "" && state.beforeDiffHash == afterDiffHash {
			if state.continueCount < maxContinues {
				state.continueCount++
				yellow.Fprintf(a.output(), "⚠️  Step %d: write tools executed but no file changes detected (%d/%d)\n",
					step.ID, state.continueCount, maxContinues)
				a.History = append(a.History, api.Message{
					Role: "user",
					Content: fmt.Sprintf("[SYSTEM] Step %d executed write tools but git diff shows no new changes. "+
						"The tool may have failed silently. Verify and retry.", step.ID),
				})
				return stepLoopContinue, nil
			}
		}
	}

	if !state.stepCompletionVerified {
		if needsContinue, feedback := a.verifyCompletionWithDiagnostics(response); needsContinue {
			state.stepCompletionVerified = true
			yellow.Fprintln(a.output(), "⚠️  Step completion verification: LSP errors found in modified files")
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: feedback,
			})
			return stepLoopContinue, nil
		}
	}

	a.printFinalAssistantResponse(response)
	green.Fprintf(a.output(), "✓ Step %d completed\n", step.ID)
	return stepLoopDone, nil
}

func (r *TurnRunner) processStepToolCalls(execToolCalls []*tools.ToolCall, step *plan.PlanStep, p *plan.Plan, idx int, rs *retryState, state *stepRunState) (bool, error) {
	a := r.agent

	r.executeToolCalls("", execToolCalls, r.planStepSkipFn, func(_ int, toolCall *tools.ToolCall, result string, change *tools.FileChange) {
		a.noteProjectMapMutation(toolCall, change)
		a.appendSessionToolExecution(toolCall, result)

		if !strings.HasPrefix(result, "Error:") &&
			!strings.HasPrefix(result, "[CANCELLED]") && !strings.HasPrefix(result, "[COMMENT]") {
			switch toolCall.Tool {
			case "str_replace":
				if path := toolCall.Args["path"]; path != "" {
					a.addPendingLSPFile(path)
				}
			case "apply_patch":
				a.addPendingLSPFilesFromChange(change)
			}
		}

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

		a.handleFileChange(change)

		if toolCall.ID != "" {
			toolMsg := api.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: toolCall.ID,
				ToolName:   toolCall.Tool,
			}
			a.History = append(a.History, toolMsg)
			a.appendSessionMessageFromAPI(toolMsg, a.CurrentModel)
		} else {
			toolResultMsg := api.Message{
				Role:    "user",
				Content: fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, result),
			}
			a.History = append(a.History, toolResultMsg)
			a.appendSessionMessage(toolResultMsg.Role, toolResultMsg.Content, a.CurrentModel)
		}
	})

	if state.lastFailedResult == "" {
		return false, nil
	}

	level := rs.recordFailure(state.lastFailedResult)
	switch level {
	case stalledNone:
		a.ui().StopSpinner()
		red.Fprintf(a.output(), "❌ Step %d Failed (auto-retry %d)\n", step.ID, rs.count)
		yellow.Fprintf(a.output(), "🔄 Retrying...\n")

		retryInstruction := planModeRetryInstruction(rs.count)
		a.History = append(a.History, api.Message{
			Role: "user",
			Content: fmt.Sprintf("The previous step FAILED with the following error:\n\n%s\n\n%s",
				state.lastFailedResult, retryInstruction),
		})
		return true, r.ExecuteStep(p, step, idx, rs)

	case stalledSoft:
		a.ui().StopSpinner()
		yellow.Fprintf(a.output(), "⚠️  Step %d: similar failure repeated %d times (auto-retry %d)\n", step.ID, rs.sameCount, rs.count)
		yellow.Fprintf(a.output(), "🔄 Retrying with strategy change...\n")

		a.History = append(a.History, api.Message{
			Role: "user",
			Content: fmt.Sprintf("The previous step FAILED with the following error:\n\n%s\n\n"+
				"WARNING: A similar failure has now occurred %d times in a row.\n"+
				"Your previous approach is likely wrong — do not repeat the same fix pattern.\n\n%s",
				state.lastFailedResult, rs.sameCount, planModeRetryInstruction(rs.count)),
		})
		return true, r.ExecuteStep(p, step, idx, rs)
	}

	a.SetStatus(StateWaitingApproval, "Step failed - waiting for action", "ステップ失敗 - アクション待ち", "Choose action", "アクションを選択")
	a.ui().StopSpinner()

	for {
		action, comment := promptFailureActionWithSelector(a.ui().PromptIO(), step, state.lastFailedResult, state.lastFailReason, rs.count)
		switch action {
		case plan.FailureActionRetry:
			a.History = append(a.History, api.Message{
				Role: "user",
				Content: fmt.Sprintf(`The previous step FAILED with the following error:

%s

Please:
1. Analyze the error carefully
2. Identify the root cause
3. Fix the code or configuration
4. Re-run the step to verify the fix

Do NOT skip this step. The issue must be resolved before proceeding.`, state.lastFailedResult),
			})
			return true, r.ExecuteStep(p, step, idx, &retryState{})
		case plan.FailureActionComment:
			a.History = append(a.History, api.Message{
				Role: "user",
				Content: fmt.Sprintf(`The previous step FAILED. Here are the user's instructions for fixing it:

%s

Error that occurred:
%s

Please follow these instructions to fix the issue and retry the step.`, comment, state.lastFailedResult),
			})
			return true, r.ExecuteStep(p, step, idx, &retryState{})
		case plan.FailureActionSkip:
			yellow.Fprintf(a.output(), "⏭️  Step %d skipped by user\n", step.ID)
			return true, nil
		case plan.FailureActionAbort:
			red.Fprintf(a.output(), "🛑 Step %d aborted by user\n", step.ID)
			return true, fmt.Errorf("step %d aborted by user: %s", step.ID, state.lastFailReason)
		}
	}
}

func (r *TurnRunner) planStepSkipFn(tc *tools.ToolCall) (bool, string) {
	if tc.Tool == "create_plan" || tc.Tool == "update_plan" {
		yellow.Fprintf(r.agent.output(), "⚠️  Ignored deprecated planning tool call: %s\n", tc.Tool)
		return true, fmt.Sprintf("[%s] Ignored: planning tools are deprecated. Continue with current step.", tc.Tool)
	}
	return false, ""
}
