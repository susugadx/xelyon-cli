package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type planModeRequest struct {
	agent               *Agent
	ctx                 context.Context
	originalUserRequest string
	preparedUserRequest string
	investigationPrompt string
	checkpoint          planModeCheckpoint
}

func newPlanModeRequest(agent *Agent, ctx context.Context, userRequest string) *planModeRequest {
	return &planModeRequest{
		agent:               agent,
		ctx:                 ctx,
		originalUserRequest: userRequest,
	}
}

func (r *planModeRequest) Run() error {
	r.prepare()
	restoreConversationOnExit := true
	restoreApprovedPlanOnExit := false
	restorePromptOnExit := true
	defer func() {
		if restoreConversationOnExit {
			var err error
			if restoreApprovedPlanOnExit {
				err = r.restoreTechnicalFailureState()
			} else {
				err = r.restoreConversationState()
			}
			if err != nil {
				red.Fprintf(r.agent.output(), "Failed to restore plan mode state: %v\n", err)
			}
			return
		}
		if restorePromptOnExit {
			r.restorePlanningPrompt()
		}
	}()

	p, handled, err := r.runInvestigation()
	if err != nil {
		restoreApprovedPlanOnExit = true
		return err
	}
	if handled {
		return err
	}
	restoreConversationOnExit = r.shouldRestoreConversationOnExit(p)
	restorePromptOnExit = !restoreConversationOnExit

	handled, err = r.handleInvestigationResult(p)
	if err != nil || handled {
		return err
	}

	return nil
}

func (r *planModeRequest) prepare() {
	r.checkpoint = capturePlanModeCheckpoint(r.agent, r.originalUserRequest)
	r.agent.clearApprovedPlanState()
	r.ensurePlanningPrompt()
	r.preparedUserRequest = r.prepareUserRequest(r.originalUserRequest)
	toolVisibility := r.agent.toolVisibilityPolicy(toolSurfacePhasePlan, toolVisibilityOptions{allowSubAgents: true})
	r.investigationPrompt = promptplan.BuildInvestigationPrompt(r.preparedUserRequest, toolVisibility.allowLowLevelInvestigation)
}

func (r *planModeRequest) ensurePlanningPrompt() {
	a := r.agent
	planningPrompt := promptplan.BuildPlanningPrompt()
	if strings.Contains(a.SystemPrompt, planningPrompt) {
		return
	}
	a.SystemPrompt = a.SystemPrompt + api.SystemPromptCacheBoundary + planningPrompt
}

func (r *planModeRequest) prepareUserRequest(userRequest string) string {
	warning := CheckBeforeImplementation(userRequest)
	if warning == "" {
		return userRequest
	}

	yellow.Fprintln(r.agent.output(), warning)
	return userRequest + "\n\n[SYSTEM NOTE: " + warning + " Please check existing code before creating new definitions.]"
}

func (r *planModeRequest) runInvestigation() (*plan.Plan, bool, error) {
	a := r.agent

	cyan.Fprintln(a.output(), "\n🔍 Investigation phase - researching the codebase...")
	a.SetStatus(StateRunning, "Investigating", "調査中", "Wait for investigation", "調査完了を待ってください")
	a.History = append(a.History, api.Message{Role: "user", Content: r.investigationPrompt})

	p, err := a.runInvestigationPhase(r.ctx)
	if err != nil {
		if r.handleTokenLimit(err) {
			return nil, true, nil
		}
		return nil, false, err
	}
	return p, false, nil
}

func (r *planModeRequest) handleInvestigationResult(p *plan.Plan) (bool, error) {
	a := r.agent
	out := a.output()

	if p == nil {
		green.Fprintln(out, "\n✓ Investigation complete. No implementation needed.")
		a.setReadyForInputStatus()
		return true, nil
	}

	if len(p.Steps) == 0 {
		green.Fprintln(out, "\n✓ Investigation complete. No implementation steps needed.")
		a.setReadyForInputStatus()
		return true, nil
	}

	planText := r.renderPlan(p)

	a.SetStatus(StateWaitingApproval, "Waiting for plan approval", "計画の承認待ち", "Answer y/n/c", "y/n/c で回答")
	approved, feedback := a.confirmPlan()
	if approved {
		a.setPendingApprovedPlan(planText)
		a.PlanModeEnabled = false
		green.Fprintln(out, "✓ Plan approved. Plan Mode complete. Implementation not started.")
		a.setReadyForInputStatus()
		return true, nil
	}

	if feedback != "" {
		yellow.Fprintf(out, "Plan rejected with feedback: %s\n", feedback)
		if err := r.restoreConversationState(); err != nil {
			return true, err
		}
		return true, r.rerunPlanMode(r.feedbackRerunRequest(feedback))
	}

	red.Fprintln(out, "Plan mode cancelled.")
	return true, nil
}

func (r *planModeRequest) renderPlan(p *plan.Plan) string {
	planDisplay := ui.NewPlanDisplay("Implementation Plan").
		SetSummary(p.Summary)

	for _, step := range p.Steps {
		planDisplay.AddStep(step.ID, step.Description, step.Tools, step.TargetFiles)
	}

	rendered := planDisplay.Render()
	_, _ = fmt.Fprintln(r.agent.output())
	_, _ = fmt.Fprint(r.agent.output(), rendered)
	return rendered
}

func (r *planModeRequest) handleTokenLimit(err error) bool {
	if !token.IsTokenLimitError(err) {
		return false
	}

	retryFunc := func() error {
		if restoreErr := r.restoreTechnicalFailureState(); restoreErr != nil {
			return restoreErr
		}
		return r.rerunPlanMode(r.originalUserRequest)
	}
	return r.agent.handleTokenLimitErrorWithRetry(err, retryFunc, true)
}

func (r *planModeRequest) feedbackRerunRequest(feedback string) string {
	base := strings.TrimSpace(r.originalUserRequest)
	if base == "" {
		return "Previous plan feedback: " + feedback
	}
	return base + " (Previous plan feedback: " + feedback + ")"
}

func (r *planModeRequest) rerunPlanMode(userRequest string) error {
	r.recordRerunRequest(userRequest)
	return r.agent.RunPlanMode(r.ctx, userRequest)
}

func (r *planModeRequest) recordRerunRequest(userRequest string) {
	if r.agent == nil || r.agent.session == nil {
		return
	}

	userRequest = strings.TrimSpace(userRequest)
	if userRequest == "" {
		return
	}

	if len(r.agent.session.Messages) > 0 {
		last := r.agent.session.Messages[len(r.agent.session.Messages)-1]
		if last.Role == "user" && strings.TrimSpace(last.Content) == userRequest {
			return
		}
	}

	r.agent.appendSessionMessage("user", userRequest, r.agent.CurrentModel)
}

func (r *planModeRequest) restoreConversationState() error {
	return r.checkpoint.restore(r.agent, false)
}

func (r *planModeRequest) restoreTechnicalFailureState() error {
	return r.checkpoint.restore(r.agent, true)
}

func (r *planModeRequest) restorePlanningPrompt() {
	r.checkpoint.restoreSystemPrompt(r.agent)
}

func (r *planModeRequest) shouldRestoreConversationOnExit(p *plan.Plan) bool {
	if p == nil {
		return false
	}
	return len(p.Steps) > 0
}
