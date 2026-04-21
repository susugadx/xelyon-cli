package agent

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	filetool "github.com/susugadx/xelyon-cli/internal/tools/file"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type ToolExecCallback func(idx int, tc *tools.ToolCall, result string, change *tools.FileChange)

type toolExecResult struct {
	result string
	change *tools.FileChange
}

// executeToolForParallel は並列実行用のツール実行関数。
// goroutine から安全に呼び出せるよう、以下を省略している:
//   - spinner / SetGlobalSpinner: goroutine 内で global spinner を操作すると競合する
//   - stdout/stderr: io.Discard を使って補助出力を抑制する
//
// stdout 出力の抑制:
//
//	tools.ExecuteQuietWithContext を使用し、wrapper 層の出力（ヘッダー・引数・折りたたみ結果）を抑制する。
//	Tool.Run() 内部の補助的な出力は execCtx.Stdout=io.Discard に流し、process stdout へは出さない。
//
// cancel の実効性（best effort）:
//
//	ctx.Err() を実行前にチェックして早期リターンする。
//	Tool.Run() まで request context を伝播するが、各ツールがそれを使うかは個別実装次第。
//	現状は bash など context-aware なツールが実行中キャンセルを拾える。
func (a *Agent) executeToolForParallel(ctx context.Context, tc *tools.ToolCall) (string, *tools.FileChange) {
	// wait_agent はリアルタイムイベント表示を使用（parallel path でも live view を優先）
	if tc.Tool == "wait_agent" {
		return a.executeWaitAgentWithLiveView(ctx, tc)
	}

	if ctx.Err() != nil {
		return "Error: context cancelled", nil
	}

	// ネガティブキャッシュチェック（ToolCache は thread-safe）
	if a.ToolCache != nil {
		if _, hit := a.ToolCache.CheckNegativeCache(tc.Tool, tc.RawArgs); hit {
			a.addOptimizationMetrics(OptimizationMetrics{NegativeCacheHits: 1})
		}
	}

	// ExecuteQuietWithContext: ヘッダー・引数・折りたたみ出力と補助 stdout を抑制（parallel path 用）
	result, change := tools.ExecuteQuietWithContext(a.toolExecutionContext(ctx, strings.NewReader(""), io.Discard, io.Discard), tc)
	a.recordToolResultOptimizations(tc, result)

	if a.ToolCache != nil {
		a.ToolCache.SetNegativeCache(tc.Tool, tc.RawArgs, result)
	}

	return result, change
}

func (a *Agent) executeReadFileBatch(ctx context.Context, paths []string) string {
	tc := buildReadFileBatchToolCall(paths, true)
	if ctx.Err() != nil {
		return "Error: context cancelled"
	}

	if a.ToolCache != nil {
		if _, hit := a.ToolCache.CheckNegativeCache(tc.Tool, tc.RawArgs); hit {
			a.addOptimizationMetrics(OptimizationMetrics{NegativeCacheHits: 1})
		}
	}

	execCtx := a.toolExecutionContext(ctx, strings.NewReader(""), io.Discard, io.Discard)
	result := filetool.ExecuteReadFilesWithLocator(execCtx.Output(), execCtx.EffectiveConfig(), execCtx.EffectiveToolCache(), paths, filetool.DefaultFullLines, execCtx.EffectiveLocatorRegistry())
	a.recordToolResultOptimizations(tc, result)

	if a.ToolCache != nil {
		a.ToolCache.SetNegativeCache(tc.Tool, tc.RawArgs, result)
	}

	return result
}

// printParallelToolGroup は非TUIモード向けに並列実行グループの結果を表示する。
func printParallelToolGroup(out io.Writer, cfg *config.Config, allToolCalls []*tools.ToolCall, indices []int, results []toolExecResult, elapsed time.Duration) {
	if len(indices) == 0 {
		return
	}

	if len(indices) == 1 {
		idx := indices[0]
		_, _ = fmt.Fprintln(out, ui.FormatToolLine(ui.ToolDisplayInfo{
			ToolName: allToolCalls[idx].Tool,
			Args:     allToolCalls[idx].Args,
			Result:   results[idx].result,
			Error:    strings.HasPrefix(strings.TrimSpace(results[idx].result), "Error:"),
		}))
		printParallelCollapsedOutput(out, cfg, allToolCalls[idx].Tool, results[idx].result)
		return
	}

	ui.PrintParallelGroupStartToWriter(out, len(indices))
	for _, idx := range indices {
		ui.PrintParallelGroupLineToWriter(out, ui.FormatToolLine(ui.ToolDisplayInfo{
			ToolName: allToolCalls[idx].Tool,
			Args:     allToolCalls[idx].Args,
			Result:   results[idx].result,
			Error:    strings.HasPrefix(strings.TrimSpace(results[idx].result), "Error:"),
		}))
		printParallelCollapsedOutputWithPrefix(out, cfg, allToolCalls[idx].Tool, results[idx].result, "│    ")
	}
	ui.PrintParallelGroupEndToWriter(out, formatParallelGroupSummary(allToolCalls, indices, elapsed))
}

func shouldShowParallelCollapsed(toolName, result string) bool {
	trimmed := strings.TrimSpace(result)
	if strings.HasPrefix(trimmed, "Error:") {
		return true
	}
	if toolName == "search_code" && strings.Contains(result, "\n") {
		return true
	}
	return false
}

func printParallelCollapsedOutput(out io.Writer, cfg *config.Config, toolName, result string) {
	if !shouldShowParallelCollapsed(toolName, result) {
		return
	}
	_, _ = fmt.Fprintln(out, ui.FormatToolOutput(result, ui.GetMaxVisibleLinesWithConfig(cfg)))
}

func printParallelCollapsedOutputWithPrefix(out io.Writer, cfg *config.Config, toolName, result, prefix string) {
	if !shouldShowParallelCollapsed(toolName, result) {
		return
	}

	collapsed := ui.FormatToolOutput(result, ui.GetMaxVisibleLinesWithConfig(cfg))
	for _, line := range strings.Split(strings.TrimRight(collapsed, "\n"), "\n") {
		if line == "" {
			continue
		}
		_, _ = fmt.Fprintf(out, "%s%s\n", prefix, line)
	}
}

func formatParallelGroupSummary(allToolCalls []*tools.ToolCall, indices []int, elapsed time.Duration) string {
	counts := make(map[string]int)
	var order []string

	addCount := func(label string) {
		if counts[label] == 0 {
			order = append(order, label)
		}
		counts[label]++
	}

	for _, idx := range indices {
		switch allToolCalls[idx].Tool {
		case "read_file", "read_files", "list_dir":
			addCount("reads")
		case "search_code":
			addCount("searches")
		case "web_search":
			addCount("web")
		case "bash":
			addCount("bash")
		default:
			addCount(allToolCalls[idx].Tool)
		}
	}

	var parts []string
	for _, label := range order {
		count := counts[label]
		switch label {
		case "reads", "searches":
			parts = append(parts, fmt.Sprintf("%d %s", count, label))
		case "web":
			parts = append(parts, fmt.Sprintf("%d web", count))
		case "bash":
			parts = append(parts, fmt.Sprintf("%d bash", count))
		default:
			if count == 1 {
				parts = append(parts, fmt.Sprintf("%d %s", count, label))
			} else {
				parts = append(parts, fmt.Sprintf("%d %ss", count, label))
			}
		}
	}

	if len(parts) == 0 {
		return fmt.Sprintf("Done (%s)", ui.FormatParallelElapsed(elapsed))
	}
	return fmt.Sprintf("Done: %s (%s)", strings.Join(parts, ", "), ui.FormatParallelElapsed(elapsed))
}

// sendParallelToolResults は TUI モードで並列実行結果を個別にチャネルへ送信する。
// ToolResultCallback と同じ二重保護（closed チェック + select default）を適用する。
func (a *Agent) sendParallelToolResults(allToolCalls []*tools.ToolCall, indices []int, results []toolExecResult, elapsed time.Duration) {
	ch := a.tuiToolResultCh
	for _, idx := range indices {
		if a.tuiToolResultClosed.Load() {
			return
		}
		tc := allToolCalls[idx]
		select {
		case ch <- tools.ToolResultInfo{
			ToolName: tc.Tool,
			Args:     tc.Args,
			Result:   results[idx].result,
			Error:    strings.HasPrefix(strings.TrimSpace(results[idx].result), "Error:"),
			Duration: elapsed,
		}:
		default:
		}
	}
}

// recordSearchCodeBatchMerge は search_code batch merge をメトリクスに記録する。
// batch merge は tool_call + tool result が個別に履歴に残るため
// API 入力トークンの削減にはならない（実行最適化のみ）。
func (a *Agent) recordSearchCodeBatchMerge(mergedCount int) {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.ToolObs.SearchCodeBatchMerges++
	}
}

// recordReadFileBatchMerge は read_file batch merge をメトリクスに記録する。
// batch merge は tool_call + tool result が個別に履歴に残るため
// API 入力トークンの削減にはならない（実行最適化のみ）。
func (a *Agent) recordReadFileBatchMerge(mergedCount int) {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.ToolObs.ReadFileBatchMerges++
	}
}

func parallelGroupSpinnerMessage(allToolCalls []*tools.ToolCall, indices []int) string {
	if len(indices) == 0 {
		return "Running parallel tools..."
	}

	counts := make(map[string]int)
	for _, idx := range indices {
		switch allToolCalls[idx].Tool {
		case "read_file", "read_files", "list_dir":
			counts["reads"]++
		case "search_code":
			counts["searches"]++
		case "web_search":
			counts["web"]++
		case "spawn_agent":
			counts["spawn"]++
		case "wait_agent":
			counts["wait"]++
		default:
			counts["tools"]++
		}
	}

	switch {
	case counts["reads"] > 0 && len(counts) == 1:
		return "Reading files..."
	case counts["searches"] > 0 && len(counts) == 1:
		return "Searching code..."
	case counts["spawn"] > 0 && len(counts) == 1:
		return fmt.Sprintf("Spawning %d sub-agents...", counts["spawn"])
	case counts["wait"] > 0 && len(counts) == 1:
		return waitAgentSpinnerMessage(allToolCalls[indices[0]].Args)
	default:
		return fmt.Sprintf("Running %d parallel tools...", len(indices))
	}
}
