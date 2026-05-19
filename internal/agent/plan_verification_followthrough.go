package agent

func (h *planModeImplementationHandoff) verificationHints() []string {
	if h == nil {
		return nil
	}

	values := make([]string, 0)
	for _, step := range h.approvedPlan.Steps {
		values = append(values, step.Verification...)
	}
	return compactHandoffValues(values, nil)
}

func (a *Agent) setTaskPlanVerification(values []string) {
	a.taskPlanVerification = compactHandoffValues(values, nil)
}
