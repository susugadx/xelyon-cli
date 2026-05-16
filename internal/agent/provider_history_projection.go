package agent

import "github.com/susugadx/xelyon-cli/internal/api"

type providerHistoryProjectionResult struct {
	History []api.Message
	Report  ProviderHistoryProjectionReport
}

func (a *Agent) providerFacingHistory() []api.Message {
	return a.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{}).History
}

func (a *Agent) providerFacingHistoryExcludingLatestMessage() []api.Message {
	history := a.providerFacingHistory()
	if len(history) == 0 {
		return nil
	}
	return history[:len(history)-1]
}

func (a *Agent) buildProviderHistoryProjection(policy ProviderHistoryReductionPolicy) providerHistoryProjectionResult {
	policy = normalizeProviderHistoryReductionPolicy(policy)
	if a == nil {
		return providerHistoryProjectionResult{
			Report: buildProviderHistoryProjectionReport(nil, nil, policy),
		}
	}
	a.historyMu.Lock()
	projection := api.CloneMessages(a.History)
	a.historyMu.Unlock()

	return providerHistoryProjectionResult{
		History: projection,
		Report:  buildProviderHistoryProjectionReport(projection, projection, policy),
	}
}
