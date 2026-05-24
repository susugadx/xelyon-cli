package agent

import "github.com/susugadx/xelyon-cli/internal/ledger"

// handleLedgerCommand は runtime task ledger の現在値を表示する。
func handleLedgerCommand(agent *Agent, args []string) bool {
	out := agent.output()
	if len(args) != 0 {
		yellow.Fprintln(out, "Usage: /ledger")
		return true
	}

	renderLedgerCommandOutput(out, ledgerCommandSnapshot(agent))
	renderLedgerRehydratePlanSection(out, agent.buildProviderHistoryRehydratePlan(nil))
	return true
}

func ledgerCommandSnapshot(agent *Agent) ledger.RuntimeTaskState {
	if agent == nil || agent.Runtime == nil || agent.Runtime.TaskLedger == nil {
		return ledger.RuntimeTaskState{}
	}
	return agent.Runtime.TaskLedger.Snapshot()
}
