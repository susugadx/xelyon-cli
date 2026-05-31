package agent

import (
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/providerhistory"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

type providerHistoryProjectionResult struct {
	History []api.Message
	Report  ProviderHistoryProjectionReport
}

func (a *Agent) providerFacingHistory() []api.Message {
	if a == nil {
		return nil
	}
	return a.providerFacingHistoryFromRaw(a.cloneRawHistoryForProviderProjection())
}

func (a *Agent) providerFacingHistoryExcludingLatestMessage() []api.Message {
	if a == nil {
		return nil
	}
	raw := a.cloneRawHistoryForProviderProjection()
	if len(raw) > 0 {
		raw = raw[:len(raw)-1]
	}
	return a.providerFacingHistoryFromRaw(raw)
}

func (a *Agent) providerFacingHistoryFromRaw(raw []api.Message) []api.Message {
	result := a.providerFacingHistoryProjectionFromRaw(raw)
	a.recordLastProviderHistoryProjectionReport(result.Report)
	return result.History
}

func (a *Agent) providerFacingHistoryProjectionFromRaw(raw []api.Message) providerHistoryProjectionResult {
	return a.buildProviderHistoryProjectionFromRaw(providerHistoryReductionPolicyForRuntime(a.Runtime), raw)
}

func (a *Agent) tokenBudgetHistory() []api.Message {
	if a == nil {
		return nil
	}
	return a.providerFacingHistoryProjectionFromRaw(a.cloneRawHistoryForProviderProjection()).History
}

func (a *Agent) buildProviderHistoryProjection(policy ProviderHistoryReductionPolicy) providerHistoryProjectionResult {
	if a == nil {
		result := providerhistory.Project(providerhistory.ProjectionInput{Policy: normalizeProviderHistoryReductionPolicy(policy)})
		return providerHistoryProjectionResult{History: result.History, Report: result.Report}
	}
	return a.buildProviderHistoryProjectionFromRaw(policy, a.cloneRawHistoryForProviderProjection())
}

func (a *Agent) cloneRawHistoryForProviderProjection() []api.Message {
	if a == nil {
		return nil
	}
	a.historyMu.Lock()
	raw := api.CloneMessages(a.History)
	a.historyMu.Unlock()
	return raw
}

func (a *Agent) buildProviderHistoryProjectionFromRaw(policy ProviderHistoryReductionPolicy, raw []api.Message) providerHistoryProjectionResult {
	result := providerhistory.Project(providerhistory.ProjectionInput{
		Messages: raw,
		Policy:   a.providerHistoryProjectionPolicy(policy),
	})
	return providerHistoryProjectionResult{
		History: result.History,
		Report:  result.Report,
	}
}

func (a *Agent) providerHistoryProjectionPolicy(policy ProviderHistoryReductionPolicy) ProviderHistoryReductionPolicy {
	policy = normalizeProviderHistoryReductionPolicy(policy)
	if policy.Mode != ProviderHistoryReductionApply {
		return policy
	}
	policy.EvidencePointers = a.providerHistoryReductionEvidencePointers()
	if a != nil && a.Runtime != nil && a.Runtime.Options.EnableProviderHistoryRehydrateContext {
		policy.EvidenceReductionRequiresActiveContext = true
		policy.ActiveContextTransportAvailable = a.providerActiveContextTransport() != api.ActiveContextTransportNone
	}
	return policy
}

func providerHistoryReductionPolicyForRuntime(runtime *AgentRuntime) ProviderHistoryReductionPolicy {
	return ProviderHistoryReductionPolicy{
		Mode: providerHistoryReductionModeResolutionForRuntime(runtime).effective,
	}
}

func (a *Agent) recordLastProviderHistoryProjectionReport(report ProviderHistoryProjectionReport) {
	if a == nil || a.Runtime == nil {
		return
	}
	a.Runtime.LastProviderHistoryProjectionReport = cloneProviderHistoryProjectionReport(report)
}

func cloneProviderHistoryProjectionReport(report ProviderHistoryProjectionReport) ProviderHistoryProjectionReport {
	return providerhistory.CloneProjectionReport(report)
}

func (a *Agent) providerHistoryReductionEvidencePointers() []taskstate.EvidencePointer {
	if a == nil || a.Runtime == nil || a.Runtime.TaskLedger == nil {
		return nil
	}
	return taskstate.EvidencePointersFromState(a.Runtime.TaskLedger.Snapshot())
}
