package agent

import "github.com/susugadx/xelyon-cli/internal/agent/plan"

func (a *Agent) setTaskPlanVerification(values []string) {
	a.taskPlanVerification = plan.CompactVerificationHints(values)
}
