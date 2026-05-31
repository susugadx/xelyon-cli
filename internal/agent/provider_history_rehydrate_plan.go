package agent

import (
	"github.com/susugadx/xelyon-cli/internal/ledger"
	"github.com/susugadx/xelyon-cli/internal/providerhistory"
)

func providerHistoryAppliedEvidencePointers(report ProviderHistoryProjectionReport) []ledger.EvidencePointer {
	return providerhistory.AppliedEvidencePointers(report)
}

func (a *Agent) buildProviderHistoryRehydratePlan(targetPaths []string) ledger.RehydratePlan {
	if a == nil || a.Runtime == nil {
		return ledger.RehydratePlan{}
	}
	return a.buildProviderHistoryRehydratePlanFromReport(a.Runtime.LastProviderHistoryProjectionReport, targetPaths)
}

func (a *Agent) buildProviderHistoryRehydratePlanFromReport(report ProviderHistoryProjectionReport, targetPaths []string) ledger.RehydratePlan {
	if a == nil || a.Runtime == nil || a.Runtime.TaskLedger == nil {
		return ledger.RehydratePlan{}
	}
	return providerhistory.BuildRehydratePlan(a.Runtime.TaskLedger, report, targetPaths)
}
