package agent

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/agent/token"
)

type planImplementationRequest struct {
	agent       *Agent
	ctx         context.Context
	userRequest string
	plan        *plan.Plan
}

func newPlanImplementationRequest(agent *Agent, ctx context.Context, userRequest string, p *plan.Plan) *planImplementationRequest {
	return &planImplementationRequest{
		agent:       agent,
		ctx:         ctx,
		userRequest: userRequest,
		plan:        p,
	}
}

func (r *planImplementationRequest) Run() error {
	if err := r.agent.runImplementationPhase(r.ctx, r.plan); err != nil {
		return r.handleFailure(err)
	}

	r.agent.runCompletionHooksWithRetry(r.ctx)
	r.agent.showTaskSummary()
	r.agent.setReadyForInputStatus()
	return nil
}

func (r *planImplementationRequest) handleFailure(err error) error {
	if r.handleTokenLimit(err) {
		return nil
	}

	r.agent.SetStatus(StateAborted, "Implementation failed", "実装に失敗", "Review errors and retry", "エラーを確認して再試行")
	return err
}

func (r *planImplementationRequest) handleTokenLimit(err error) bool {
	if !token.IsTokenLimitError(err) {
		return false
	}

	retryFunc := func() error {
		return r.agent.RunPlanMode(r.ctx, r.userRequest)
	}
	return r.agent.handleTokenLimitErrorWithRetry(err, retryFunc, true)
}
