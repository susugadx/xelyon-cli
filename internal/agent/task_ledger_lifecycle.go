package agent

func (a *Agent) resetProviderFacingTaskLedger() {
	if a == nil || a.Runtime == nil || a.Runtime.TaskLedger == nil {
		return
	}
	if !a.providerRequestsConsumeTaskLedger() {
		return
	}
	a.Runtime.TaskLedger.Reset()
}

func (a *Agent) providerRequestsConsumeTaskLedger() bool {
	if a == nil || a.Runtime == nil {
		return false
	}
	return a.shouldSendActiveContextToProvider() ||
		providerHistoryReductionPolicyForRuntime(a.Runtime).Mode == ProviderHistoryReductionApply
}
