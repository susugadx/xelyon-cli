package agent

import (
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type parallelResultDelivery struct {
	agent     *Agent
	state     *toolruntime.ParallelCallState
	callback  ToolExecCallback
	threshold int
}

func newParallelResultDelivery(agent *Agent, state *toolruntime.ParallelCallState, callback ToolExecCallback) *parallelResultDelivery {
	return &parallelResultDelivery{
		agent:     agent,
		state:     state,
		callback:  callback,
		threshold: agent.cfg().LoopDetection.Threshold,
	}
}

// deliverToolExecutionResults は Phase2 の結果配送（history 反映 + callback 呼び出し）を担当する。
func (a *Agent) deliverToolExecutionResults(state *toolruntime.ParallelCallState, callback ToolExecCallback) {
	newParallelResultDelivery(a, state, callback).deliverAll()
}

func (d *parallelResultDelivery) deliverAll() {
	for i, tc := range d.state.AllToolCalls {
		d.deliverAt(i, tc, d.state.Entries[i])
	}
}

func (d *parallelResultDelivery) deliverAt(idx int, tc *tools.ToolCall, entry toolruntime.ParallelCallEntry) {
	switch entry.Status {
	case toolruntime.ParallelCallStatusSkip:
		d.agent.appendToolResultToHistory(tc, entry.SkipMsg)
	case toolruntime.ParallelCallStatusLoopAbort:
		d.appendLoopAbortHistory(tc, idx)
	case toolruntime.ParallelCallStatusBatched, toolruntime.ParallelCallStatusExecute:
		d.deliverExecutedToolResult(idx, tc, d.state.Results[idx])
	}
}

func (d *parallelResultDelivery) appendLoopAbortHistory(tc *tools.ToolCall, idx int) {
	msg, ok := toolruntime.BuildLoopAbortHistoryMessage(tc, idx, d.state.LoopTriggerIdx, d.threshold)
	if !ok {
		return
	}
	d.agent.History = append(d.agent.History, msg)
}

func (d *parallelResultDelivery) deliverExecutedToolResult(idx int, tc *tools.ToolCall, result toolruntime.Result) {
	if d.agent.Stats != nil {
		d.agent.Stats.AddToolExecution(tc.Tool)
	}
	if d.callback != nil {
		d.callback(idx, tc, result)
	}
}
