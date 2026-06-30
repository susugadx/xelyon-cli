package agent

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/providerhistory"
)

func (a *Agent) providerFacingHistoryForRequest(ctx context.Context) (context.Context, []api.Message) {
	if a == nil {
		return ctx, nil
	}
	return a.providerFacingHistoryForRequestFromRaw(ctx, a.cloneRawHistoryForProviderProjection())
}

func (a *Agent) providerFacingHistoryForRequestFromRaw(ctx context.Context, raw []api.Message) (context.Context, []api.Message) {
	result, rawOutputContext := a.providerHistoryProjectionForRequest(ctx, raw)
	a.recordLastProviderHistoryProjectionReport(result.Report)
	if providerhistory.ProjectionDisablesResponseIDChain(result.Report) {
		ctx = api.WithResponseIDChainDisabled(ctx)
		a.clearResponseContextForProviderHistoryReductionRequest()
	}
	ctx = appendProviderHistoryRehydratedEvidenceActiveContext(ctx, a.providerHistoryRehydratedEvidenceActiveContextBlocks(ctx, result.Report))
	ctx = appendProviderHistoryRawOutputActiveContext(ctx, rawOutputContext.Blocks)
	return ctx, result.History
}

func (a *Agent) providerHistoryProjectionForRequest(ctx context.Context, raw []api.Message) (providerHistoryProjectionResult, providerHistoryRawOutputActiveContextBuild) {
	result := a.providerFacingHistoryProjectionFromRaw(raw)
	rawOutputContext := a.buildProviderHistoryRawOutputActiveContext(ctx, result.Report, raw)
	if rawOutputContext.missingRequiredRefs() {
		result = a.buildProviderHistoryProjectionFromRawWithRawOutputApplyDisabledReason(raw, rawOutputContext.failClosedReason())
		rawOutputContext = a.buildProviderHistoryRawOutputActiveContext(ctx, result.Report, raw)
	}
	return result, rawOutputContext
}

func (a *Agent) providerHistoryProjectionForTokenBudget(ctx context.Context, raw []api.Message) (providerHistoryProjectionResult, providerHistoryRawOutputActiveContextBuild) {
	result := a.buildProviderHistoryProjectionFromRawForTokenBudget(raw)
	rawOutputContext := a.buildProviderHistoryRawOutputActiveContext(ctx, result.Report, raw)
	if rawOutputContext.missingRequiredRefs() {
		result = a.buildProviderHistoryProjectionFromRawForTokenBudget(raw)
		rawOutputContext = a.buildProviderHistoryRawOutputActiveContext(ctx, result.Report, raw)
	}
	return result, rawOutputContext
}

func (a *Agent) clearResponseContextForProviderHistoryReductionRequest() {
	if !a.hasResponseIDChainProvider() {
		return
	}
	a.clearProviderAndSavedResponseContext()
}
