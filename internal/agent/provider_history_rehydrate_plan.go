package agent

import (
	"github.com/susugadx/xelyon-cli/internal/providerhistory"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

func providerHistoryAppliedEvidencePointers(report ProviderHistoryProjectionReport) []taskstate.EvidencePointer {
	return providerhistory.AppliedEvidencePointers(report)
}

func (a *Agent) buildProviderHistoryRehydratePlan(targetPaths []string) taskstate.RehydratePlan {
	if a == nil || a.Runtime == nil {
		return taskstate.RehydratePlan{}
	}
	return a.buildProviderHistoryRehydratePlanFromReport(a.Runtime.LastProviderHistoryProjectionReport, targetPaths)
}

func (a *Agent) buildProviderHistoryRehydratePlanFromReport(report ProviderHistoryProjectionReport, targetPaths []string) taskstate.RehydratePlan {
	if a == nil || a.Runtime == nil || a.Runtime.TaskLedger == nil {
		return taskstate.RehydratePlan{}
	}
	return providerhistory.BuildRehydratePlan(a.Runtime.TaskLedger, report, targetPaths)
}
