package agent

import (
	"context"
	"sync"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

type parallelToolCallStatus int

const (
	parallelToolCallStatusExecute parallelToolCallStatus = iota
	parallelToolCallStatusSkip
	parallelToolCallStatusLoopAbort
	parallelToolCallStatusBatched
)

type parallelToolCallEntry struct {
	status  parallelToolCallStatus
	skipMsg string
}

type parallelToolCallState struct {
	allToolCalls []*tools.ToolCall
	entries      []parallelToolCallEntry
	results      []toolExecResult

	parallelEntries   []int
	sequentialEntries []int

	loopTriggerIdx int
	loopDetected   bool
}

func newParallelToolCallState(allToolCalls []*tools.ToolCall) *parallelToolCallState {
	return &parallelToolCallState{
		allToolCalls:   allToolCalls,
		entries:        make([]parallelToolCallEntry, len(allToolCalls)),
		results:        make([]toolExecResult, len(allToolCalls)),
		loopTriggerIdx: -1,
	}
}

// executeToolCallsWithParallel は parallel-safe なツールを並列実行し、sequential なツールを順次実行する。
// 通常モードと Plan Mode の両方で使用する共通 executor。
//
// NOTE: Investigation Phase（plan_investigation.go）は本 executor を使用しない。
// Investigation Phase は SafetyHigh 制約のある独自ループ（executeToolOnly）を持ち、
// 並列実行の対象外である。
//
// 処理フロー:
//
//	Phase 0: 実行前フィルタリング（loopDetectFn → skipFn の順で評価）
//	Phase 0.5: search_code / read_file batch 最適化
//	Phase 1: 実行（parallel-safe は goroutine、sequential は順次）
//	Phase 2: 結果配送（history 反映と callback 呼び出し）
//
// loop detection 履歴メッセージは旧 shouldAbortToolLoopWithResponse と意味的に同等の
// フォーマットを維持する（FC trigger/tool skip/text-based trigger の仕様を保持）。
func (a *Agent) executeToolCallsWithParallel(
	ctx context.Context,
	allToolCalls []*tools.ToolCall,
	loopDetectFn func(tc *tools.ToolCall) (abort bool),
	skipFn func(tc *tools.ToolCall) (skip bool, msg string),
	callback ToolExecCallback,
) (loopDetected bool) {
	state := newParallelToolCallState(allToolCalls)
	if len(state.allToolCalls) == 0 {
		return false
	}

	a.planParallelToolCalls(state, loopDetectFn, skipFn)
	a.executeSearchCodeBatch(ctx, state)
	a.executeReadFileBatchMerge(ctx, state)
	a.partitionParallelAndSequential(state)
	a.runParallelSafeToolsPhase(ctx, state)
	a.executeSequentialTools(ctx, state)
	a.deliverToolExecutionResults(state, callback)

	return state.loopDetected
}

// planParallelToolCalls は Phase0 の判定を担当し、各 tool call の実行状態を確定する。
// loopDetectFn を skipFn より先に評価し、旧 sequential 実装と同じ連続性判定契約を維持する。
func (a *Agent) planParallelToolCalls(
	state *parallelToolCallState,
	loopDetectFn func(tc *tools.ToolCall) (abort bool),
	skipFn func(tc *tools.ToolCall) (skip bool, msg string),
) {
	aborted := false
	for i, tc := range state.allToolCalls {
		if aborted {
			state.entries[i] = parallelToolCallEntry{status: parallelToolCallStatusLoopAbort}
			continue
		}
		if loopDetectFn != nil && loopDetectFn(tc) {
			state.entries[i] = parallelToolCallEntry{status: parallelToolCallStatusLoopAbort}
			state.loopDetected = true
			state.loopTriggerIdx = i
			aborted = true
			continue
		}
		if skipFn != nil {
			if skip, msg := skipFn(tc); skip {
				state.entries[i] = parallelToolCallEntry{status: parallelToolCallStatusSkip, skipMsg: msg}
				continue
			}
		}
		state.entries[i] = parallelToolCallEntry{status: parallelToolCallStatusExecute}
	}
}

func (a *Agent) partitionParallelAndSequential(state *parallelToolCallState) {
	state.parallelEntries = state.parallelEntries[:0]
	state.sequentialEntries = state.sequentialEntries[:0]

	for i, entry := range state.entries {
		if entry.status != parallelToolCallStatusExecute {
			continue
		}
		if tools.IsParallelSafe(state.allToolCalls[i]) {
			state.parallelEntries = append(state.parallelEntries, i)
		} else {
			state.sequentialEntries = append(state.sequentialEntries, i)
		}
	}
}

func (a *Agent) runParallelSafeToolsPhase(ctx context.Context, state *parallelToolCallState) {
	if len(state.parallelEntries) == 0 {
		return
	}

	parallelSpinner := a.ui().NewSpinner()
	parallelSpinner.Start(parallelGroupSpinnerMessage(state.allToolCalls, state.parallelEntries))
	a.ui().SetSpinner(parallelSpinner)
	// panic 経路でも spinner が残留しないよう、最終的な停止を保証する。
	defer a.ui().StopSpinner()

	elapsed := a.executeParallelSafeTools(ctx, state)
	// 非TUI出力と競合しないよう、結果表示前に停止する。
	a.ui().StopSpinner()
	a.reportParallelSafeExecution(state, elapsed)
}

// executeParallelSafeTools は並列安全ツール群のワーカー実行のみを担当する。
// spinner 制御や表示は runParallelSafeToolsPhase 側で行う。
func (a *Agent) executeParallelSafeTools(ctx context.Context, state *parallelToolCallState) time.Duration {
	startedAt := time.Now()
	sem := make(chan struct{}, tools.MaxParallelTools)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, idx := range state.parallelEntries {
		if ctx.Err() != nil {
			mu.Lock()
			state.results[idx] = toolExecResult{result: "Error: context cancelled"}
			mu.Unlock()
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()

			r, c := a.executeToolForParallel(ctx, state.allToolCalls[i])
			mu.Lock()
			state.results[i] = toolExecResult{result: r, change: c}
			mu.Unlock()
		}(idx)
	}
	wg.Wait()
	return time.Since(startedAt)
}

func (a *Agent) reportParallelSafeExecution(state *parallelToolCallState, elapsed time.Duration) {
	if a.tuiToolResultCh != nil {
		a.sendParallelToolResults(state.allToolCalls, state.parallelEntries, state.results, elapsed)
		return
	}
	printParallelToolGroup(a.output(), a.cfg(), state.allToolCalls, state.parallelEntries, state.results, elapsed)
}

func (a *Agent) executeSequentialTools(ctx context.Context, state *parallelToolCallState) {
	for _, idx := range state.sequentialEntries {
		if ctx.Err() != nil {
			state.results[idx] = toolExecResult{result: "Error: context cancelled"}
			continue
		}
		r, c := a.executeToolWithSpinner(ctx, state.allToolCalls[idx])
		state.results[idx] = toolExecResult{result: r, change: c}
	}
}
