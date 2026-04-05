package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
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
	completionVerified    bool
	hookRetryCount        int
	textPlanRedirectCount int
	fallbackResponse      string
	reachedHardLimit      bool
}

type normalModeAction int

const (
	normalModeContinue normalModeAction = iota
	normalModeBreak
	normalModeDone
)

func (r *TurnRunner) runNormalModeLoop(input string, image *api.ImageData) error {
	a := r.agent

	normalModeInput := input + promptnormal.NormalModePrompt
	a.History = append(a.History, api.Message{Role: "user", Content: normalModeInput})

	cfg := a.cfg()
	hardLimit := normalizeToolLoopLimit(cfg.General.ToolLoopLimit)
	state := &normalModeState{}

	for i := 0; ; i++ {
		if hardLimit > 0 && i >= hardLimit {
			state.reachedHardLimit = true
			break
		}
		if hardLimit == 0 {
			emitLoopWarning(a, i)
		}

		response, err := r.requestNormalModeResponse(input, image, i)
		if err != nil {
			return err
		}

		toolCalls := r.prepareToolCalls(response)
		action, handled, err := r.handlePlanJSONFallback(response, toolCalls)
		if err != nil {
			return err
		}
		if handled {
			if action == normalModeDone {
				return nil
			}
			continue
		}

		r.debugLogToolCalls(response, toolCalls)

		if len(toolCalls) == 0 {
			action := r.handleNormalModeNoToolResponse(response, cfg, state)
			if action == normalModeContinue {
				continue
			}
			if action == normalModeBreak {
				break
			}
			if action == normalModeDone {
				a.showTaskSummary()
				return nil
			}
		}

		a.maybePrintAssistantPhaseUpdate(response, execToolCallsSummaryInput(toolCalls))

		if err := r.processNormalModeToolCalls(response, toolCalls, &state.rs); err != nil {
			return err
		}
	}

	if state.reachedHardLimit {
		yellow.Fprintf(a.output(), "⚠️  Tool loop limit reached (%d iterations)\n", hardLimit)
	}
	if state.fallbackResponse != "" {
		a.printFinalAssistantResponse(state.fallbackResponse)
	}
	a.showTaskSummary()
	return nil
}

func (r *TurnRunner) requestNormalModeResponse(input string, image *api.ImageData, iteration int) (string, error) {
	a := r.agent
	if iteration == 0 {
		a.refreshProjectPromptIfDirty(input)
	}
	effectivePrompt := prompt.StripPlanningReferences(a.SystemPrompt)

	requestCtx := a.requestContext(r.ctx)
	if iteration == 0 && image != nil {
		inputWithPrompt := input + promptnormal.NormalModePrompt
		response, err := a.CurrentProvider.ChatWithImage(
			requestCtx, effectivePrompt, a.History[:len(a.History)-1], inputWithPrompt, image, a.CurrentModel,
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

func (r *TurnRunner) handlePlanJSONFallback(response string, toolCalls []*tools.ToolCall) (normalModeAction, bool, error) {
	if len(toolCalls) != 0 || !plan.ContainsPlanJSON(response) {
		return normalModeContinue, false, nil
	}

	a := r.agent
	if planJSON := plan.ExtractPlanJSON(response); planJSON != "" {
		if p, err := plan.ParsePlan(planJSON); err == nil && len(p.Steps) > 0 {
			yellow.Fprintf(a.output(), "📋 FC fallback: extracted %d-step plan from text. Switching to step-by-step...\n", len(p.Steps))
			r.appendAssistantHistoryOnly(response)
			if err := a.runImplementationPhase(r.ctx, p); err != nil {
				return normalModeContinue, true, err
			}
			a.runCompletionHooksWithRetry(r.ctx)
			a.showTaskSummary()
			return normalModeDone, true, nil
		}
	}

	yellow.Fprintln(a.output(), "⚠️  Plan JSON detected but parse failed. Execute tools directly.")
	r.appendAssistantHistoryOnly(response)
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: "[SYSTEM] You are in NORMAL MODE. Do NOT output JSON directly. Execute the required changes directly using tools (read_file, str_replace, etc).",
	})
	return normalModeContinue, true, nil
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
	a := r.agent
	steps := extractTextPlan(response)
	if !containsCompletionDeclaration(response) && len(steps) >= 5 && isActionPlan(steps) {
		state.textPlanRedirectCount++
		if state.textPlanRedirectCount > maxTextPlanHardLimit {
			yellow.Fprintf(a.output(), "⚠️  Text plan detected %d times without tool use. Returning response to user.\n", state.textPlanRedirectCount)
			r.appendAssistantHistoryOnly(response)
			state.fallbackResponse = response
			return normalModeBreak
		}

		if state.textPlanRedirectCount > maxTextPlanRedirects {
			yellow.Fprintf(a.output(), "⚠️  Text plan detected %d times. Forcing direct execution.\n", state.textPlanRedirectCount)
			r.appendAssistantHistoryOnly(response)
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: "[SYSTEM] STOP planning. Pick the FIRST change and execute it NOW using the appropriate tool (read_file, str_replace, etc). One tool call, no explanation.",
			})
			return normalModeContinue
		}

		yellow.Fprintf(a.output(), "⚠️  Text plan detected (%d steps). Execute tools directly instead. (%d/%d)\n",
			len(steps), state.textPlanRedirectCount, maxTextPlanRedirects)
		r.appendAssistantHistoryOnly(response)
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: "[SYSTEM] Do NOT output plans as numbered text. Execute the required changes directly using tools (read_file, str_replace, etc).",
		})
		return normalModeContinue
	}

	if !state.completionVerified {
		needsContinue, feedback := a.verifyCompletionWithDiagnostics(response)
		if needsContinue {
			state.completionVerified = true
			yellow.Fprintln(a.output(), "⚠️  Completion verification: LSP errors found in modified files")
			r.appendAssistantHistoryOnly(response)
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: feedback,
			})
			return normalModeContinue
		}
	}

	if containsCompletionDeclaration(response) && len(cfg.Hooks.OnCompletion) > 0 {
		changedFiles := a.getTaskChangedFiles()
		if len(changedFiles) > 0 {
			hookNeedsContinue, hookFeedback := a.checkGitDiffEmpty()
			if !hookNeedsContinue {
				hookNeedsContinue, hookFeedback = a.runCompletionHooks(changedFiles)
			}
			if hookNeedsContinue {
				state.hookRetryCount++
				maxRetry := cfg.Hooks.MaxRetry
				if maxRetry <= 0 {
					maxRetry = 3
				}
				if state.hookRetryCount >= maxRetry {
					yellow.Fprintf(a.output(), "⚠️  Hook retry limit reached (%d/%d). Proceeding with completion.\n", state.hookRetryCount, maxRetry)
				} else {
					yellow.Fprintf(a.output(), "⚠️  Completion hook failed (%d/%d). Asking AI to fix...\n", state.hookRetryCount, maxRetry)
					r.appendAssistantHistoryOnly(response)
					a.History = append(a.History, api.Message{
						Role:    "user",
						Content: hookFeedback,
					})
					return normalModeContinue
				}
			}
		}
	}

	a.handleNormalResponse(response)
	return normalModeDone
}

func (r *TurnRunner) processNormalModeToolCalls(response string, toolCalls []*tools.ToolCall, rs *retryState) error {
	a := r.agent
	var lastFailedResult string

	execToolCalls := r.processDeprecatedCreatePlanCalls(response, toolCalls)
	if len(execToolCalls) > 0 {
		a.addToolCallsToHistory(response, execToolCalls)
	}

	toolLoopDetected := r.executeToolCalls(response, execToolCalls, nil, func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		a.noteProjectMapMutation(tc, change)
		a.appendSessionToolExecution(tc, result)

		if a.handleStrReplaceErrors(tc, result) {
			return
		}
		if a.handleCommentFlow(tc, result) {
			return
		}

		a.handleFileChange(change)

		if tc.ID != "" {
			toolMsg := api.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
				ToolName:   tc.Tool,
			}
			a.History = append(a.History, toolMsg)
			a.appendSessionMessageFromAPI(toolMsg, a.CurrentModel)
		} else {
			toolResultMsg := api.Message{
				Role:    "user",
				Content: fmt.Sprintf("[Tool Result for %s]\n%s", tc.Tool, result),
			}
			a.History = append(a.History, toolResultMsg)
			a.appendSessionMessage(toolResultMsg.Role, toolResultMsg.Content, a.CurrentModel)
		}
		_, _ = fmt.Fprintln(a.output())

		if !strings.HasPrefix(result, "Error:") &&
			!strings.HasPrefix(result, "[CANCELLED]") && !strings.HasPrefix(result, "[COMMENT]") {
			switch tc.Tool {
			case "str_replace":
				if path := tc.Args["path"]; path != "" {
					a.addPendingLSPFile(path)
				}
			case "apply_patch":
				a.addPendingLSPFilesFromChange(change)
			}
		}

		if tc.Tool == "bash" || tools.IsWriteTool(tc.Tool) {
			if failed, _ := plan.ContainsFailure(result); failed {
				lastFailedResult = result
			}
		}
	})
	if toolLoopDetected {
		return fmt.Errorf("tool loop detected")
	}

	if lastFailedResult != "" {
		level := rs.recordFailure(lastFailedResult)

		switch level {
		case stalledNone:
			a.ui().ResetTerminalState()
			red.Fprintf(a.output(), "❌ Failed (retry %d)\n", rs.count)
			yellow.Fprintf(a.output(), "🔄 Retrying...\n")

			retryInstruction := normalModeRetryInstruction(rs.count)
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: fmt.Sprintf("The previous tool execution FAILED (attempt %d):\n\n%s\n\n%s", rs.count, lastFailedResult, retryInstruction),
			})
			return nil

		case stalledSoft:
			a.ui().ResetTerminalState()
			yellow.Fprintf(a.output(), "⚠️  Similar failure repeated %d times (retry %d)\n", rs.sameCount, rs.count)
			yellow.Fprintf(a.output(), "🔄 Retrying with strategy change...\n")

			a.History = append(a.History, api.Message{
				Role: "user",
				Content: fmt.Sprintf("The previous tool execution FAILED (attempt %d):\n\n%s\n\n"+
					"WARNING: A similar failure has now occurred %d times in a row.\n"+
					"Your previous approach is likely wrong — do not repeat the same fix pattern.\n\n%s",
					rs.count, lastFailedResult, rs.sameCount, normalModeRetryInstruction(rs.count)),
			})
			return nil
		}

		a.ui().ResetTerminalState()
		red.Fprintf(a.output(), "❌ Stalled — same error %d times\n", rs.sameCount)
		yellow.Fprintln(a.output(), "Could not complete the task automatically. Letting AI respond...")
		rs.reset()
		return nil
	}

	if rs.count > 0 {
		green.Fprintf(a.output(), "✅ Succeeded (on retry %d)\n", rs.count)
		rs.reset()
	}
	return nil
}

func (r *TurnRunner) processDeprecatedCreatePlanCalls(response string, toolCalls []*tools.ToolCall) []*tools.ToolCall {
	a := r.agent
	execToolCalls := make([]*tools.ToolCall, 0, len(toolCalls))

	for _, toolCall := range toolCalls {
		if toolCall.Tool != "create_plan" {
			execToolCalls = append(execToolCalls, toolCall)
			continue
		}

		if a.Stats != nil {
			a.Stats.AddToolExecution(toolCall.Tool)
		}
		result, _ := a.executeToolWithSpinner(r.ctx, toolCall)
		a.appendSessionToolExecution(toolCall, result)
		r.appendAssistantHistoryOnly(response)
		a.appendSessionMessage("assistant", response, a.CurrentModel)

		toolResultMsg := api.Message{
			Role:    "user",
			Content: fmt.Sprintf("[Tool Result for create_plan]\n%s", result),
		}
		a.History = append(a.History, toolResultMsg)
		a.appendSessionMessage(toolResultMsg.Role, toolResultMsg.Content, a.CurrentModel)

		yellow.Fprintln(a.output(), "⚠️  create_plan is deprecated, continuing in normal mode...")
	}

	return execToolCalls
}
