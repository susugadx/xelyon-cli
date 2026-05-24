package agent

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ledger"
)

const providerHistoryRehydratedEvidenceActiveContextName = "provider_history_rehydrated_evidence"

func (a *Agent) providerHistoryRehydratedEvidenceActiveContextBlock(ctx context.Context, report ProviderHistoryProjectionReport) (api.ActiveContextBlock, bool) {
	if !a.shouldBuildProviderHistoryRehydratedEvidenceActiveContext() {
		return api.ActiveContextBlock{}, false
	}

	plan := a.buildProviderHistoryRehydratePlanFromReport(report, nil)
	if len(plan.Items) == 0 {
		return api.ActiveContextBlock{}, false
	}

	executionReport := a.Runtime.TaskLedger.ExecuteRehydratePlan(ctx, plan, ledger.RehydratePlanExecutionOptions{})
	content := ledger.RenderRehydratedEvidenceBlock(executionReport.Block)
	if strings.TrimSpace(content) == "" {
		return api.ActiveContextBlock{}, false
	}
	return api.ActiveContextBlock{
		Name:    providerHistoryRehydratedEvidenceActiveContextName,
		Content: content,
	}, true
}

func (a *Agent) providerHistoryRehydratedEvidenceActiveContextBlocks(ctx context.Context, report ProviderHistoryProjectionReport) []api.ActiveContextBlock {
	block, ok := a.providerHistoryRehydratedEvidenceActiveContextBlock(ctx, report)
	if !ok {
		return nil
	}
	return []api.ActiveContextBlock{block}
}

func (a *Agent) providerFacingActiveContextBlocksForTokenBudget(ctx context.Context) []api.ActiveContextBlock {
	blocks := a.providerFacingActiveContextBlocks()
	if !a.shouldBuildProviderHistoryRehydratedEvidenceActiveContext() {
		return blocks
	}
	result := a.providerFacingHistoryProjectionFromRaw(a.cloneRawHistoryForProviderProjection())
	blocks = append(blocks, a.providerHistoryRehydratedEvidenceActiveContextBlocks(ctx, result.Report)...)
	return blocks
}

func (a *Agent) shouldBuildProviderHistoryRehydratedEvidenceActiveContext() bool {
	if a == nil || a.Runtime == nil || a.Runtime.TaskLedger == nil {
		return false
	}
	return a.Runtime.Options.EnableProviderHistoryRehydrateContext && a.providerCanConsumeActiveContext()
}

func appendProviderHistoryRehydratedEvidenceActiveContext(ctx context.Context, additions []api.ActiveContextBlock) context.Context {
	if len(additions) == 0 {
		return ctx
	}
	blocks := api.ActiveContextBlocksFromContext(ctx)
	blocks = append(blocks, additions...)
	return api.WithActiveContextBlocks(ctx, blocks)
}
