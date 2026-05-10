package toolruntime

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ParallelCallStatus は parallel tool call plan における各 call の状態を表す。
type ParallelCallStatus int

const (
	// ParallelCallStatusExecute は tool call を実行対象にする状態。
	ParallelCallStatusExecute ParallelCallStatus = iota
	// ParallelCallStatusSkip は tool call を実行せず skip 結果を履歴へ返す状態。
	ParallelCallStatusSkip
	// ParallelCallStatusLoopAbort は loop detection により実行しない状態。
	ParallelCallStatusLoopAbort
	// ParallelCallStatusBatched は batch 実行済み結果を個別 call へ配送する状態。
	ParallelCallStatusBatched
)

// ParallelCallEntry は tool call 1 件分の実行 plan を表す。
type ParallelCallEntry struct {
	Status  ParallelCallStatus
	SkipMsg string
}

// Result は tool call 実行結果と file change をまとめる。
type Result struct {
	Result      string
	Change      *tools.FileChange
	Observation *tools.RuntimeObservation
	Error       bool
}

// ParallelCallState は 1 turn 内の tool call 実行 plan と結果を保持する。
type ParallelCallState struct {
	AllToolCalls []*tools.ToolCall
	Entries      []ParallelCallEntry
	Results      []Result

	ParallelEntries   []int
	SequentialEntries []int

	LoopTriggerIdx int
	LoopDetected   bool
}

// NewParallelCallState は tool call 一覧から初期状態を作る。
func NewParallelCallState(allToolCalls []*tools.ToolCall) *ParallelCallState {
	return &ParallelCallState{
		AllToolCalls:   allToolCalls,
		Entries:        make([]ParallelCallEntry, len(allToolCalls)),
		Results:        make([]Result, len(allToolCalls)),
		LoopTriggerIdx: -1,
	}
}

// PlanParallelCalls は loop detection と skip policy を評価して各 call の状態を確定する。
func PlanParallelCalls(
	state *ParallelCallState,
	loopDetectFn func(tc *tools.ToolCall) (abort bool),
	skipFn func(tc *tools.ToolCall) (skip bool, msg string),
) {
	aborted := false
	for i, tc := range state.AllToolCalls {
		if aborted {
			state.Entries[i] = ParallelCallEntry{Status: ParallelCallStatusLoopAbort}
			continue
		}
		if loopDetectFn != nil && loopDetectFn(tc) {
			state.Entries[i] = ParallelCallEntry{Status: ParallelCallStatusLoopAbort}
			state.LoopDetected = true
			state.LoopTriggerIdx = i
			aborted = true
			continue
		}
		if skipFn != nil {
			if skip, msg := skipFn(tc); skip {
				state.Entries[i] = ParallelCallEntry{Status: ParallelCallStatusSkip, SkipMsg: msg}
				continue
			}
		}
		state.Entries[i] = ParallelCallEntry{Status: ParallelCallStatusExecute}
	}
}

// PartitionParallelAndSequential は実行対象 call を parallel-safe と sequential に分ける。
func PartitionParallelAndSequential(state *ParallelCallState) {
	state.ParallelEntries = state.ParallelEntries[:0]
	state.SequentialEntries = state.SequentialEntries[:0]

	for i, entry := range state.Entries {
		if entry.Status != ParallelCallStatusExecute {
			continue
		}
		if tools.IsParallelSafe(state.AllToolCalls[i]) {
			state.ParallelEntries = append(state.ParallelEntries, i)
		} else {
			state.SequentialEntries = append(state.SequentialEntries, i)
		}
	}
}

// BuildLoopAbortHistoryMessage は loop abort された tool call の履歴 message を作る。
func BuildLoopAbortHistoryMessage(tc *tools.ToolCall, idx, loopTriggerIdx, threshold int) (api.Message, bool) {
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
