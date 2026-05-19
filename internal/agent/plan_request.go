package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
)

type planModeRequest struct {
	agent               *Agent
	ctx                 context.Context
	originalUserRequest string
	preparedUserRequest string
	investigationPrompt string
	checkpoint          planModeCheckpoint
	approved            bool
	handoff             *planModeImplementationHandoff
	autoCompression     *autoCompressionTurnState
}

type planModeRequestOptions struct {
	autoCompression *autoCompressionTurnState
}

type planModeRestoreMode int

const (
	planModeRestoreConversation planModeRestoreMode = iota
	planModeRestorePromptOnly
	planModeRestoreNone
)

type planModeExitPolicy struct {
	restoreMode          planModeRestoreMode
	clearResponseContext bool
}

type planModeInvestigationOutcome struct {
	plan    *plan.Plan
	handled bool
}

func newPlanModeRequest(agent *Agent, ctx context.Context, userRequest string) *planModeRequest {
	return newPlanModeRequestWithOptions(agent, ctx, userRequest, planModeRequestOptions{})
}

func newPlanModeRequestWithOptions(agent *Agent, ctx context.Context, userRequest string, opts planModeRequestOptions) *planModeRequest {
	return &planModeRequest{
		agent:               agent,
		ctx:                 ctx,
		originalUserRequest: userRequest,
		autoCompression:     opts.autoCompression,
	}
}

func (r *planModeRequest) Run() error {
	r.prepare()
	exitPolicy := planModeExitPolicy{restoreMode: planModeRestoreConversation}
	defer func() {
		r.applyExitPolicy(exitPolicy)
	}()

	outcome, err := r.executeInvestigationStep()
	if err != nil {
		return err
	}
	if outcome.handled {
		return nil
	}
	exitPolicy = r.transitionExitPolicyAfterInvestigation(exitPolicy, outcome.plan)

	handled, err := r.executeApprovalStep(outcome.plan, &exitPolicy)
	if err != nil || handled {
		return err
	}

	return nil
}

func (r *planModeRequest) executeInvestigationStep() (planModeInvestigationOutcome, error) {
	p, handled, err := r.runInvestigation()
	if err != nil {
		return planModeInvestigationOutcome{}, err
	}
	return planModeInvestigationOutcome{plan: p, handled: handled}, nil
}

func (r *planModeRequest) transitionExitPolicyAfterInvestigation(policy planModeExitPolicy, p *plan.Plan) planModeExitPolicy {
	policy.restoreMode = r.resolveExitRestoreMode(p)
	return policy
}

func (r *planModeRequest) executeApprovalStep(p *plan.Plan, policy *planModeExitPolicy) (bool, error) {
	handled, err := r.handleInvestigationResult(p)
	if !r.approved || policy == nil {
		return handled, err
	}

	// Plan 承認後は調査フェーズ履歴を通常実装ターンへ持ち越さない。
	policy.restoreMode = planModeRestoreConversation
	policy.clearResponseContext = true
	return handled, err
}

func (r *planModeRequest) applyExitPolicy(policy planModeExitPolicy) {
	if r == nil || r.agent == nil {
		return
	}

	switch policy.restoreMode {
	case planModeRestoreConversation:
		if err := r.restoreConversationState(); err != nil {
			red.Fprintf(r.agent.output(), "Failed to restore plan mode state: %v\n", err)
		}
	case planModeRestorePromptOnly:
		r.restorePlanningPrompt()
	case planModeRestoreNone:
	}

	if policy.clearResponseContext {
		r.clearResponseContext()
	}
}

func (r *planModeRequest) prepare() {
	r.checkpoint = capturePlanModeCheckpoint(r.agent, r.originalUserRequest)
	r.ensurePlanningPrompt()
	r.preparedUserRequest = r.prepareUserRequest(r.originalUserRequest)
	toolVisibility := r.agent.toolVisibilityPolicy(toolSurfacePhasePlan, toolVisibilityOptions{allowSubAgents: true})
	r.investigationPrompt = promptplan.BuildInvestigationPrompt(r.preparedUserRequest, toolVisibility.investigationSurface)
}

func (r *planModeRequest) ensurePlanningPrompt() {
	a := r.agent
	planningPrompt := promptplan.BuildPlanningPrompt()
	layout := api.SplitSystemPromptLayout(a.SystemPrompt)
	if strings.Contains(layout.Static, planningPrompt) || strings.Contains(layout.Dynamic, planningPrompt) {
		return
	}
	layout.AppendDynamic(planningPrompt)
	a.SystemPrompt = layout.Compose()
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

	p, err := a.runInvestigationPhaseWithOptions(r.ctx, planInvestigationOptions{
		autoCompression: r.investigationAutoCompression(),
	})
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
		if display := buildPlanNoImplementationDisplay(p); display != nil {
			fmt.Fprint(out, display.Render())
		}
		green.Fprintln(out, "\n✓ Investigation complete. No implementation steps needed.")
		a.setReadyForInputStatus()
		return true, nil
	}

	r.renderPlan(p)

	a.SetStatus(StateWaitingApproval, "Waiting for plan review", "計画レビュー待ち", "Approve/request changes/cancel", "承認/変更依頼/中止を選択")
	approved, feedback := r.confirmPlanApproval()
	if approved {
		r.approved = true
		r.handoff = newPlanModeImplementationHandoff(r.originalUserRequest, p)
		green.Fprintln(out, "✓ Plan approved. Plan Mode complete.")
		r.exitPlanModeReview()
		return true, nil
	}

	if feedback != "" {
		yellow.Fprintf(out, "Plan feedback received. Regenerating plan: %s\n", feedback)
		if err := r.restoreConversationState(); err != nil {
			return true, err
		}
		return true, r.rerunPlanMode(r.feedbackRerunRequest(feedback))
	}

	red.Fprintln(out, "Plan mode cancelled. No implementation started.")
	r.exitPlanModeReview()
	return true, nil
}

func (r *planModeRequest) exitPlanModeReview() {
	if r == nil || r.agent == nil {
		return
	}
	r.agent.setPlanModeEnabled(false)
	r.agent.setReadyForInputStatus()
}

func (r *planModeRequest) renderPlan(p *plan.Plan) string {
	planDisplay := buildPlanReviewDisplay(p)
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
		if restoreErr := r.restoreConversationState(); restoreErr != nil {
			return restoreErr
		}
		return r.rerunPlanMode(r.originalUserRequest)
	}
	return r.agent.handleTokenLimitErrorWithRetryOptions(err, retryFunc, tokenLimitRetryOptions{skipCompressionPersistence: true})
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
	handoff, err := r.agent.runPlanModeWithAutoCompression(r.ctx, userRequest, r.autoCompression)
	if handoff != nil {
		r.handoff = handoff
	}
	return err
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
	return r.checkpoint.restore(r.agent)
}

func (r *planModeRequest) restorePlanningPrompt() {
	r.checkpoint.restoreSystemPrompt(r.agent)
}

func (r *planModeRequest) clearResponseContext() {
	if r == nil || r.agent == nil {
		return
	}

	// 承認後の通常実装ターンは planning チェーン(previous_response_id)を
	// 継続せず、承認済み plan handoff のローカル履歴から開始する。
	r.agent.clearResponseContinuationContext()
}

func (r *planModeRequest) shouldRestoreConversationOnExit(p *plan.Plan) bool {
	if p == nil {
		return false
	}
	return len(p.Steps) > 0
}

func (r *planModeRequest) resolveExitRestoreMode(p *plan.Plan) planModeRestoreMode {
	if r.shouldRestoreConversationOnExit(p) {
		return planModeRestoreConversation
	}
	return planModeRestorePromptOnly
}
