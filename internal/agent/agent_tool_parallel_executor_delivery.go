package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type parallelResultDelivery struct {
	agent     *Agent
	state     *parallelToolCallState
	callback  ToolExecCallback
	threshold int
}

func newParallelResultDelivery(agent *Agent, state *parallelToolCallState, callback ToolExecCallback) *parallelResultDelivery {
	return &parallelResultDelivery{
		agent:     agent,
		state:     state,
		callback:  callback,
		threshold: agent.cfg().LoopDetection.Threshold,
	}
}

// deliverToolExecutionResults は Phase2 の結果配送（history 反映 + callback 呼び出し）を担当する。
func (a *Agent) deliverToolExecutionResults(state *parallelToolCallState, callback ToolExecCallback) {
	newParallelResultDelivery(a, state, callback).deliverAll()
}

func (d *parallelResultDelivery) deliverAll() {
	for i, tc := range d.state.allToolCalls {
		d.deliverAt(i, tc, d.state.entries[i])
	}
}

func (d *parallelResultDelivery) deliverAt(idx int, tc *tools.ToolCall, entry parallelToolCallEntry) {
	switch entry.status {
	case parallelToolCallStatusSkip:
		d.agent.appendToolResultToHistory(tc, entry.skipMsg)
	case parallelToolCallStatusLoopAbort:
		d.appendLoopAbortHistory(tc, idx)
	case parallelToolCallStatusBatched, parallelToolCallStatusExecute:
		d.deliverExecutedToolResult(idx, tc, d.state.results[idx])
	}
}

func (d *parallelResultDelivery) appendLoopAbortHistory(tc *tools.ToolCall, idx int) {
	msg, ok := buildLoopAbortHistoryMessage(tc, idx, d.state.loopTriggerIdx, d.threshold)
	if !ok {
		return
	}
	d.agent.History = append(d.agent.History, msg)
}

func (d *parallelResultDelivery) deliverExecutedToolResult(idx int, tc *tools.ToolCall, result toolExecResult) {
	if d.agent.Stats != nil {
		d.agent.Stats.AddToolExecution(tc.Tool)
	}
	if d.callback != nil {
		d.callback(idx, tc, result.result, result.change)
	}
}

func buildLoopAbortHistoryMessage(tc *tools.ToolCall, idx, loopTriggerIdx, threshold int) (api.Message, bool) {
	if idx == loopTriggerIdx {
		if tc.ID != "" {
			return api.Message{
				Role:       "tool",
				Content:    fmt.Sprintf("[SYSTEM] Tool loop detected: %s was called %d times. Stopping to prevent infinite loop.", tc.Tool, threshold),
				ToolCallID: tc.ID,
				ToolName:   tc.Tool,
			}, true
		}
		return api.Message{
			Role:    "user",
			Content: fmt.Sprintf("[SYSTEM WARNING] The same tool call was repeated %d times. Please try a different approach or ask the user for clarification.", threshold),
		}, true
	}

	if tc.ID == "" {
		return api.Message{}, false
	}
	return api.Message{
		Role:       "tool",
		Content:    "[SYSTEM] Skipped due to tool loop detection.",
		ToolCallID: tc.ID,
		ToolName:   tc.Tool,
	}, true
}
