package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
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

func (a *Agent) executeSearchCodeBatch(ctx context.Context, state *parallelToolCallState) {
	type searchBatchGroup struct {
		indices  []int
		patterns []string
	}

	searchGroups := make(map[string]*searchBatchGroup)
	var searchGroupOrder []string

	for i, entry := range state.entries {
		if entry.status != parallelToolCallStatusExecute {
			continue
		}
		tc := state.allToolCalls[i]
		if tc.Tool != "search_code" || !isSimpleSearchPattern(tc.Args["pattern"]) {
			continue
		}
		optKey := searchCodeOptionsKey(tc)
		if group, ok := searchGroups[optKey]; ok {
			group.indices = append(group.indices, i)
			group.patterns = append(group.patterns, tc.Args["pattern"])
		} else {
			searchGroups[optKey] = &searchBatchGroup{
				indices:  []int{i},
				patterns: []string{tc.Args["pattern"]},
			}
			searchGroupOrder = append(searchGroupOrder, optKey)
		}
	}

	for _, optKey := range searchGroupOrder {
		group := searchGroups[optKey]
		if len(group.indices) < 2 {
			continue
		}
		for chunkStart := 0; chunkStart < len(group.indices); chunkStart += maxSearchBatchPatterns {
			chunkEnd := chunkStart + maxSearchBatchPatterns
			if chunkEnd > len(group.indices) {
				chunkEnd = len(group.indices)
			}
			chunkIndices := group.indices[chunkStart:chunkEnd]
			chunkPatterns := group.patterns[chunkStart:chunkEnd]

			if len(chunkIndices) < 2 {
				continue
			}

			leaderTC := state.allToolCalls[chunkIndices[0]]
			mergedTC := cloneToolCallWithNewPattern(leaderTC, strings.Join(chunkPatterns, ","))
			mergedResult, _ := a.executeToolForParallel(ctx, mergedTC)

			perPattern := splitMultiPatternResult(mergedResult, chunkPatterns)
			if perPattern == nil {
				continue
			}

			for j, idx := range chunkIndices {
				pattern := chunkPatterns[j]
				if section, ok := perPattern[pattern]; ok {
					state.results[idx] = toolExecResult{result: section}
				} else {
					state.results[idx] = toolExecResult{result: mergedResult}
				}
				state.entries[idx] = parallelToolCallEntry{status: parallelToolCallStatusBatched}
			}
			a.recordSearchCodeBatchMerge(len(chunkIndices))
		}
	}
}

func (a *Agent) executeReadFileBatchMerge(ctx context.Context, state *parallelToolCallState) {
	execFlags := make([]bool, len(state.allToolCalls))
	for i, entry := range state.entries {
		execFlags[i] = entry.status == parallelToolCallStatusExecute
	}
	readBatchSegments := segmentReadFileBatches(state.allToolCalls, execFlags)

	for _, seg := range readBatchSegments {
		for callStart, pathStart := 0, 0; callStart < len(seg.indices); {
			chunkCallStart := callStart
			chunkPathStart := pathStart
			chunkPathCount := 0

			for callStart < len(seg.indices) {
				callPathCount := seg.pathCounts[callStart]
				if chunkPathCount > 0 && chunkPathCount+callPathCount > maxReadFileBatchPaths {
					break
				}
				chunkPathCount += callPathCount
				pathStart += callPathCount
				callStart++
			}

			chunkIndices := seg.indices[chunkCallStart:callStart]
			chunkPathCounts := seg.pathCounts[chunkCallStart:callStart]
			chunkPaths := seg.paths[chunkPathStart : chunkPathStart+chunkPathCount]

			if len(chunkIndices) < 2 {
				continue
			}

			mergedResult := a.executeReadFileBatch(ctx, chunkPaths)
			perFile := splitReadFileBatchResult(mergedResult, chunkPaths)
			if perFile == nil {
				continue
			}

			offset := 0
			for j, idx := range chunkIndices {
				callPaths := chunkPaths[offset : offset+chunkPathCounts[j]]
				offset += chunkPathCounts[j]
				if section, ok := joinReadFileBatchSections(perFile, callPaths); ok {
					state.results[idx] = toolExecResult{result: section}
				} else {
					state.results[idx] = toolExecResult{result: mergedResult}
				}
				state.entries[idx] = parallelToolCallEntry{status: parallelToolCallStatusBatched}
			}
			a.recordReadFileBatchMerge(len(chunkIndices))
		}
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
