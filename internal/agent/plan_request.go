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
	userRequest         string
	investigationPrompt string
}

func newPlanModeRequest(agent *Agent, ctx context.Context, userRequest string) *planModeRequest {
	return &planModeRequest{
		agent:       agent,
		ctx:         ctx,
		userRequest: userRequest,
	}
}

func (r *planModeRequest) Run() error {
	r.prepare()

	p, handled, err := r.runInvestigation()
	if err != nil || handled {
		return err
	}

	handled, err = r.handleInvestigationResult(p)
	if err != nil || handled {
		return err
	}

	return r.runImplementation(p)
}

func (r *planModeRequest) prepare() {
	r.ensurePlanningPrompt()
	r.userRequest = r.prepareUserRequest(r.userRequest)
	toolVisibility := r.agent.toolVisibilityPolicy(toolSurfacePhasePlan, toolVisibilityOptions{allowSubAgents: true})
	r.investigationPrompt = promptplan.BuildInvestigationPrompt(r.userRequest, toolVisibility.investigationSurface)
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

	r.renderPlan(p)

	a.SetStatus(StateWaitingApproval, "Waiting for plan approval", "計画の承認待ち", "Answer y/n/c", "y/n/c で回答")
	approved, feedback := a.confirmPlan()
	if approved {
		green.Fprintln(out, "✓ Plan approved. Starting implementation...")
		a.SetStatus(StateRunning, "Implementing", "実装中", "Wait for completion", "完了を待ってください")
		return false, nil
	}

	if feedback != "" {
		yellow.Fprintf(out, "Plan rejected with feedback: %s\n", feedback)
		return true, a.RunPlanMode(r.ctx, r.userRequest+" (Previous plan feedback: "+feedback+")")
	}

	red.Fprintln(out, "Plan execution cancelled.")
	return true, nil
}

func (r *planModeRequest) renderPlan(p *plan.Plan) {
	planDisplay := ui.NewPlanDisplay("Implementation Plan").
		SetSummary(p.Summary)

	for _, step := range p.Steps {
		planDisplay.AddStep(step.ID, step.Description, step.Tools, step.TargetFiles)
	}

	_, _ = fmt.Fprintln(r.agent.output())
	_, _ = fmt.Fprint(r.agent.output(), planDisplay.Render())
}

func (r *planModeRequest) runImplementation(p *plan.Plan) error {
	return newPlanImplementationRequest(r.agent, r.ctx, r.userRequest, p).Run()
}

func (r *planModeRequest) handleTokenLimit(err error) bool {
	if !token.IsTokenLimitError(err) {
		return false
	}

	retryFunc := func() error {
		return r.agent.RunPlanMode(r.ctx, r.userRequest)
	}
	return r.agent.handleTokenLimitErrorWithRetry(err, retryFunc, true)
}
