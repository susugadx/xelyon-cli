package agent

import "github.com/susugadx/xelyon-cli/internal/api"

type providerHistoryProjectionPolicy interface {
	ProjectProviderHistory(*Agent) []api.Message
}

type defaultProviderHistoryProjectionPolicy struct{}

func (defaultProviderHistoryProjectionPolicy) ProjectProviderHistory(a *Agent) []api.Message {
	if a == nil {
		return nil
	}
	a.historyMu.Lock()
	defer a.historyMu.Unlock()
	return api.CloneMessages(a.History)
}

func (a *Agent) providerFacingHistory() []api.Message {
	return a.buildProviderHistoryProjection(defaultProviderHistoryProjectionPolicy{})
}

func (a *Agent) providerFacingHistoryExcludingLatestMessage() []api.Message {
	history := a.providerFacingHistory()
	if len(history) == 0 {
		return nil
	}
	return history[:len(history)-1]
}

func (a *Agent) buildProviderHistoryProjection(policy providerHistoryProjectionPolicy) []api.Message {
	if a == nil || policy == nil {
		return nil
	}
	return policy.ProjectProviderHistory(a)
}
