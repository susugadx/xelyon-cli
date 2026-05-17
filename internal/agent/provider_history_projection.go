package agent

import "github.com/susugadx/xelyon-cli/internal/api"

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
	result := a.buildProviderHistoryProjectionFromRaw(providerHistoryReductionPolicyForRuntime(a.Runtime), raw)
	a.recordLastProviderHistoryProjectionReport(result.Report)
	return result.History
}

func (a *Agent) buildProviderHistoryProjection(policy ProviderHistoryReductionPolicy) providerHistoryProjectionResult {
	if a == nil {
		return providerHistoryProjectionResult{
			Report: buildProviderHistoryProjectionReport(nil, nil, normalizeProviderHistoryReductionPolicy(policy)),
		}
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
	policy = normalizeProviderHistoryReductionPolicy(policy)
	projection := raw
	if policy.Mode == ProviderHistoryReductionApply && len(raw) > 0 {
		projection = api.CloneMessages(raw)
		report := buildProviderHistoryReductionDetectionReport(raw, projection, policy.Mode)
		a.applyProviderHistoryReduction(&report, projection)
		finalizeProviderHistoryProjectionReport(&report, raw, projection)
		return providerHistoryProjectionResult{
			History: projection,
			Report:  report,
		}
	}

	return providerHistoryProjectionResult{
		History: projection,
		Report:  buildProviderHistoryProjectionReport(raw, projection, policy),
	}
}

func providerHistoryReductionPolicyForRuntime(runtime *AgentRuntime) ProviderHistoryReductionPolicy {
	if runtime == nil || !runtime.Options.EnableProviderHistoryReduction {
		return ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionDisabled}
	}
	return ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply}
}

func (a *Agent) recordLastProviderHistoryProjectionReport(report ProviderHistoryProjectionReport) {
	if a == nil || a.Runtime == nil {
		return
	}
	a.Runtime.LastProviderHistoryProjectionReport = cloneProviderHistoryProjectionReport(report)
}

func cloneProviderHistoryProjectionReport(report ProviderHistoryProjectionReport) ProviderHistoryProjectionReport {
	if len(report.Candidates) > 0 {
		report.Candidates = append([]ProviderHistoryReductionCandidate(nil), report.Candidates...)
	}
	if len(report.Kept) > 0 {
		report.Kept = append([]ProviderHistoryReductionCandidate(nil), report.Kept...)
	}
	return report
}
