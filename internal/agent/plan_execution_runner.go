package agent

import (
	"context"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type implementationPhaseRunner struct {
	agent *Agent
	ctx   context.Context
	plan  *plan.Plan
}

func newImplementationPhaseRunner(agent *Agent, ctx context.Context, p *plan.Plan) *implementationPhaseRunner {
	return &implementationPhaseRunner{
		agent: agent,
		ctx:   ctx,
		plan:  p,
	}
}

func (r *implementationPhaseRunner) Run() error {
	for {
		nextID, step, ok := r.nextStep()
		if !ok {
			break
		}

		if err := r.runStep(nextID, step); err != nil {
			return err
		}
		if r.plan.IsCompleted() {
			break
		}
	}

	return r.finish()
}

func (r *implementationPhaseRunner) nextStep() (int, *plan.PlanStep, bool) {
	nextID := r.plan.GetNextStep()
	if nextID == -1 {
		return 0, nil, false
	}

	step := r.plan.GetStep(nextID)
	if step == nil {
		return 0, nil, false
	}

	return nextID, step, true
}

func (r *implementationPhaseRunner) runStep(stepID int, step *plan.PlanStep) error {
	a := r.agent

	_, _ = fmt.Fprintf(a.output(), "\n%s\n", ui.FormatStepProgress(stepID, len(r.plan.Steps), step.Description, "running"))
	if err := a.executeStepV2(r.ctx, r.plan, step, stepID-1, &retryState{}); err != nil {
		return err
	}

	r.plan.UpdateStatus(stepID, "completed", "")
	r.runStepCompleteHooks(stepID, step)
	return nil
}

func (r *implementationPhaseRunner) runStepCompleteHooks(stepID int, step *plan.PlanStep) {
	a := r.agent
	if hooks := a.cfg().Hooks; len(hooks.OnStepComplete) > 0 {
		if !a.runStepCompleteHooksWithRetry(r.ctx, stepID, step.Description, "completed") {
			yellow.Fprintf(a.output(), "⚠️  Step %d hooks failed but proceeding to next step\n", stepID)
		}
	}
}

func (r *implementationPhaseRunner) finish() error {
	green.Fprintf(r.agent.output(), "\n✓ All %d steps completed!\n", len(r.plan.Steps))
	return r.persistSession()
}

func (r *implementationPhaseRunner) persistSession() error {
	a := r.agent
	if a.storage == nil || a.session == nil {
		return nil
	}

	a.syncResponseIDToSession()
	if err := a.storage.Save(a.session); err != nil {
		yellow.Fprintf(a.output(), "Warning: Failed to save session: %v\n", err)
	}
	return nil
}
