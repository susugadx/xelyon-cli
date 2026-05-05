package agent

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// confirmPlan は計画の承認確認
func (a *Agent) confirmPlan(ctx context.Context) (approved bool, feedback string) {
	result := tools.ConfirmInteractiveWithIO(a.requestPromptIO(ctx), "Approve this plan?")

	switch result.Action {
	case "yes":
		return true, ""
	case "no":
		return false, ""
	case "comment":
		return false, strings.TrimSpace(result.Comment)
	default:
		return false, ""
	}
}

func (r *planModeRequest) confirmPlanApproval() (approved bool, feedback string) {
	ctx, cleanup := r.agent.beginRequestPromptCancellationScope(r.ctx)
	defer cleanup()
	return r.agent.confirmPlan(ctx)
}
