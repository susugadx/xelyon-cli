package agent

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func (a *Agent) providerFacingHistoryForRequest(ctx context.Context) (context.Context, []api.Message) {
	if a == nil {
		return ctx, nil
	}
	return a.providerFacingHistoryForRequestFromRaw(ctx, a.cloneRawHistoryForProviderProjection())
}

func (a *Agent) providerFacingHistoryExcludingLatestMessageForRequest(ctx context.Context) (context.Context, []api.Message) {
	if a == nil {
		return ctx, nil
	}
	raw := a.cloneRawHistoryForProviderProjection()
	if len(raw) > 0 {
		raw = raw[:len(raw)-1]
	}
	return a.providerFacingHistoryForRequestFromRaw(ctx, raw)
}

func (a *Agent) providerFacingHistoryForRequestFromRaw(ctx context.Context, raw []api.Message) (context.Context, []api.Message) {
	result := a.providerFacingHistoryProjectionFromRaw(raw)
	a.recordLastProviderHistoryProjectionReport(result.Report)
	if providerHistoryProjectionDisablesResponseIDChain(result.Report) {
		ctx = api.WithResponseIDChainDisabled(ctx)
		a.clearResponseContextForProviderHistoryReductionRequest()
	}
	return ctx, result.History
}

func providerHistoryProjectionDisablesResponseIDChain(report ProviderHistoryProjectionReport) bool {
	return report.Mode == ProviderHistoryReductionApply && report.ReplacedCount > 0
}

func (a *Agent) clearResponseContextForProviderHistoryReductionRequest() {
	if !a.hasResponseIDChainProvider() {
		return
	}
	a.clearProviderAndSavedResponseContext()
}
