package agent

import (
	"context"
	"sync"
	"time"

	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

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
	state := toolruntime.NewParallelCallState(allToolCalls)
	if len(state.AllToolCalls) == 0 {
		return false
	}

	toolruntime.PlanParallelCalls(state, loopDetectFn, skipFn)
	batchStartedAt := time.Now()
	a.executeSearchCodeBatch(ctx, state)
	a.executeReadFileBatchMerge(ctx, state)
	a.reportBatchedTUIToolResults(state, time.Since(batchStartedAt))
	toolruntime.PartitionParallelAndSequential(state)
	a.runParallelSafeToolsPhase(ctx, state)
	a.executeSequentialTools(ctx, state)
	a.deliverToolExecutionResults(state, callback)

	return state.LoopDetected
}

func (a *Agent) runParallelSafeToolsPhase(ctx context.Context, state *toolruntime.ParallelCallState) {
	if len(state.ParallelEntries) == 0 {
		return
	}

	a.ui().StartSpinner(parallelGroupSpinnerMessage(state.AllToolCalls, state.ParallelEntries))
	// panic 経路でも spinner が残留しないよう、最終的な停止を保証する。
	defer a.ui().StopSpinner()

	elapsed := a.executeParallelSafeTools(ctx, state)
	// 非TUI出力と競合しないよう、結果表示前に停止する。
	a.ui().StopSpinner()
	a.reportParallelSafeExecution(state, elapsed)
}

// executeParallelSafeTools は並列安全ツール群のワーカー実行のみを担当する。
// spinner 制御や表示は runParallelSafeToolsPhase 側で行う。
func (a *Agent) executeParallelSafeTools(ctx context.Context, state *toolruntime.ParallelCallState) time.Duration {
	startedAt := time.Now()
	sem := make(chan struct{}, tools.MaxParallelTools)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, idx := range state.ParallelEntries {
		if ctx.Err() != nil {
			mu.Lock()
			state.Results[idx] = toolruntime.Result{Result: "Error: context cancelled", Error: true}
			mu.Unlock()
			continue
		}
		a.emitTUIToolRunning(state.AllToolCalls[idx])
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()

			execResult := a.executeToolForParallelResult(ctx, state.AllToolCalls[i])
			mu.Lock()
			state.Results[i] = toolruntime.Result{
				Result:      execResult.Result,
				Change:      execResult.Change,
				Observation: execResult.Observation,
				Error:       execResult.Error,
			}
			mu.Unlock()
		}(idx)
	}
	wg.Wait()
	return time.Since(startedAt)
}

func (a *Agent) reportParallelSafeExecution(state *toolruntime.ParallelCallState, elapsed time.Duration) {
	if a.tuiToolResultCh != nil {
		a.sendParallelToolResults(state.AllToolCalls, state.ParallelEntries, state.Results, elapsed)
		return
	}
	printParallelToolGroup(a.output(), a.cfg(), state.AllToolCalls, state.ParallelEntries, state.Results, elapsed)
}

func (a *Agent) executeSequentialTools(ctx context.Context, state *toolruntime.ParallelCallState) {
	for _, idx := range state.SequentialEntries {
		if ctx.Err() != nil {
			state.Results[idx] = toolruntime.Result{Result: "Error: context cancelled", Error: true}
			continue
		}
		execResult := a.executeToolWithSpinnerResult(ctx, state.AllToolCalls[idx])
		state.Results[idx] = toolruntime.Result{
			Result:      execResult.Result,
			Change:      execResult.Change,
			Observation: execResult.Observation,
			Error:       execResult.Error,
		}
	}
}
